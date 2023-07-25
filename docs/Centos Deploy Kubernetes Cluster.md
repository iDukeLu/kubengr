# CentOS 部署 K8S 集群
工程化想法
- 每个节点安装命令行工具，通过命令初始化 Master、Worker 节点
- 要解决的问题
	- Worker 节点如何判断 join 到哪里？手动指定 Master IP OR 附带加密字符串，包含 IP、Token 信息
	- 一些参数如何指定？比如 yum 仓库地址、docker 镜像地址
		- docker、kubeadm yum 源、docker 镜像地址
	- 如何保证安装的幂等性？失败了可以重复执行命令
	- 支持选择 CRI、CNI
	- 支持常见软件的安装，如 Ingress Controller
	- 支持指定版本的安装
	- 性能考虑？Master、Worker 节点部分步骤同步进行，如环境准备
- 在用户目录下创建 .kubengr 的隐藏文件夹
	- 会下载 kubengr.sh 文件到该目录
	- kubengr.yaml 文件保存默认配置（yum 仓库、docker 镜像、cgroup）
	- 保存修改后的 kubeadm 配置文件
	- kubengr.log 文件记录所有日志
	- 命令行可以选择默认配置和自定义配置（可以选择仓库、镜像、插件等），默认配置通过配置文件读取，自定义配置也可通过文件指定
	- .kubengr 被删除后可以自动回复，在每次执行命令前检查
- 安装完成后，返回封装的 join 命令，便于 Worker 节点 join

## 更换 CentOS 源
```shell
mv /etc/yum.repos.d/CentOS-Base.repo /etc/yum.repos.d/CentOS-Base.repo.backup
curl -o /etc/yum.repos.d/CentOS-Base.repo https://mirrors.aliyun.com/repo/Centos-7.repo
yum makecache
sed -i -e '/mirrors.cloud.aliyuncs.com/d' -e '/mirrors.aliyuncs.com/d' /etc/yum.repos.d/CentOS-Base.repo
```

## 环境准备
```shell
## 设置各节点时间精确同步
yum install -y chrony
systemctl start chronyd
systemctl enable chronyd
chronyc sources

## 安装 DNS 解析服务
yum install -y systemd-resolved 
systemctl start systemd-resolved 
systemctl enable systemd-resolved

## 关闭 firewalld 防火墙
systemctl stop firewalld 
systemctl disable firewalld
firewall-cmd --state

## 关闭 SElinux 安全模组
setenforce 0
sed -i "s/SELINUX=enforcing$/SELINUX=disabled/g" /etc/selinux/config
sed -i "s/SELINUX=permissive$/SELINUX=disabled/g" /etc/selinux/config
getenforce

## 关闭 Swap 交换分区
swapoff -a
sed -i "s/\/dev\/mapper\/centos-swap/\#\/dev\/mapper\/centos-swap/g" /etc/fstab
free -m

## 修改 iptable 桥接规则
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF
modprobe br_netfilter
sysctl --system

## 加载 IPVS 内核模块
cat > /etc/sysconfig/modules/ipvs.modules <<EOF
#!/bin/bash
modprobe -- ip_vs
modprobe -- ip_vs_rr
modprobe -- ip_vs_wrr
modprobe -- ip_vs_sh
modprobe -- nf_conntrack_ipv4
EOF
chmod 755 /etc/sysconfig/modules/ipvs.modules && bash /etc/sysconfig/modules/ipvs.modules && lsmod | grep -e ip_vs -e nf_conntrack_ipv4

## 安装一些基础工具
yum install -y ipset ipvsadm bind-utils net-tools bash-completion

## 修改主机名
HOST_NAME="kube-master"
IP_ADDRESS="127.0.0.1"
hostnamectl set-hostname $HOST_NAME
if ! grep -q "${IP_ADDRESS}.*${HOST_NAME}" /etc/hosts; then
    echo "${IP_ADDRESS} ${HOST_NAME}" >> /etc/hosts
fi
```

## 安装 Containerd
```shell
## 设置仓库
yum install -y yum-utils
yum-config-manager --add-repo \
https://download.docker.com/linux/centos/docker-ce.repo

## 安装
yum install -y containerd.io

## 修改配置文件
containerd config default > /etc/containerd/config.toml
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/g' /etc/containerd/config.toml
sed -i '/^[[:space:]]*\[plugins\."io\.containerd\.grpc\.v1\.cri"\.registry\.mirrors]$/a \
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"] \
          endpoint = ["https://12k5hoeq.mirror.aliyuncs.com"] \
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."k8s.gcr.io"] \
          endpoint = ["https://registry.aliyuncs.com/k8sxio"]' /etc/containerd/config.toml

## 启动
systemctl start containerd
systemctl enable containerd
```

## kukelet & kubeadm & kubectl
```shell
## 添加仓库
cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://mirrors.aliyun.com/kubernetes/yum/repos/kubernetes-el7-x86_64/
enabled=1
gpgcheck=0
repo_gpgcheck=0
gpgkey=https://mirrors.aliyun.com/kubernetes/yum/doc/yum-key.gpg https://mirrors.aliyun.com/kubernetes/yum/doc/rpm-package-key.gpg
EOF

## 安装 kukelet & kubeadm & kubectl
yum install -y kubelet kubeadm kubectl


## 修改 kukelet 启动参数
cat << EOF > /etc/sysconfig/kubelet
KUBELET_EXTRA_ARGS="--cgroup-driver=systemd --container-runtime=remote --container-runtime-endpoint=unix:///run/containerd/containerd.sock"
EOF

## 启动
systemctl start kubelet
systemctl enable kubelet
```

## 初始化 Master
```shell
## 提前拉取镜像
kubeadm config images pull \
--image-repository=registry.cn-hangzhou.aliyuncs.com/google_containers

## 集群初始化
kubeadm init --kubernetes-version=$(kubelet --version | awk '{print $2}') \
--pod-network-cidr=10.244.0.0/16 \
--image-repository=registry.cn-hangzhou.aliyuncs.com/google_containers

## 写入配置
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
export KUBECONFIG=/etc/kubernetes/admin.conf

## 去除污点（可选）
kubectl taint nodes turing node-role.kubernetes.io/master:NoSchedule-
```

## CNI（仅 Master）

```shell
kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml
```
参考：https://github.com/flannel-io/flannel#deploying-flannel-manually

## 集群加入节点
```shell
## 加入 worker 节点
sudo kubeadm token create --print-join-command --ttl=0
```