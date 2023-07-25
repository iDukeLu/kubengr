package scripts

const Script = `
echo_red() {
  echo -e "\033[31m $1 \033[0m"
}

echo_green() {
  echo -e "\033[32m $1 \033[0m"
}

echo_yellow() {
  echo -e "\033[33m $1 \033[0m"
}

## 更换 CentOS yum 源
set_aliyun_yum_repo() {
    base_repo="/etc/yum.repos.d/CentOS-Base.repo"
    expected_mirror="mirrors.aliyun.com"

    if grep -q "$expected_mirror" "$base_repo"; then
        yum repolist
        echo_green "切换阿里云 yum 源...ok"
    fi

    repo_file="/etc/yum.repos.d/CentOS-Base.repo"
    backup_repo_file="/etc/yum.repos.d/CentOS-Base.repo.backup"
    if [ ! -f "$backup_repo_file" ]; then
        mv "$repo_file" "$backup_repo_file"
    fi

    curl -o /etc/yum.repos.d/CentOS-Base.repo https://mirrors.aliyun.com/repo/Centos-7.repo
    sed -i -e '/mirrors.cloud.aliyuncs.com/d' -e '/mirrors.aliyuncs.com/d' /etc/yum.repos.d/CentOS-Base.repo
    yum makecache

    yum repolist
    echo_green "切换阿里云 yum 源...ok"
}


## 修改主机名
set_host_name() {
    host_name=$1
    ip_address="127.0.0.1"

    hostnamectl set-hostname "$host_name"

    if ! grep -q "${ip_address}.*${host_name}" /etc/hosts; then
        echo "${ip_address}   ${host_name}" >> /etc/hosts
    fi
    echo_green "设置主机名 $host_name...ok"
}


## 设置各节点时间精确同步
synchronous_clock() {
    yum install chrony -y
    systemctl start chronyd && systemctl enable chronyd
    chronyc sources
    echo_green "同步服务器时间...ok"
}


## 开启 DNS 解析服务
enable_resolution() {
    yum install -y systemd-resolved
    systemctl start systemd-resolved && systemctl enable systemd-resolved

    if (systemctl is-active --quiet systemd-resolved) && (systemctl is-enabled --quiet systemd-resolved); then
        echo_green "启动/启用 DNS 解析服务...ok"
    else
        echo_red "启动/启用 DNS 解析服务...failed"
        return 1
    fi
}


## 禁用 firewalld 防火墙
disable_firewalld() {
    systemctl stop firewalld && systemctl disable firewalld
    
    firewalld_active=$(systemctl is-active firewalld 2>/dev/null)
    firewalld_enabled=$(systemctl is-enabled firewalld 2>/dev/null)
    if [ "$firewalld_enabled" != "enabled" ] && [ "$firewalld_active" != "active" ]; then
        firewall-cmd --state
        echo_green "关闭/禁用 firewalld 防火墙...ok"
    else
        echo_red "关闭/禁用 firewalld 防火墙...failed"
    fi
}


## 关闭 SElinux 安全模组
close_selinux() {
    selinux_state=$(getenforce)
    setenforce 0
    if [ "$selinux_state" = "Enforcing" ]; then
        sed -i "s/SELINUX=enforcing$/SELINUX=disabled/g" /etc/selinux/config
    fi
    if [ "$selinux_state" = "Permissive" ]; then
        sed -i "s/SELINUX=permissive$/SELINUX=disabled/g" /etc/selinux/config
    fi
    selinux_state=$(getenforce)
    if [ "$selinux_state" = "Permissive" ] || [ "$selinux_state" = "Disabled" ]; then
        echo_green "关闭 SElinux 安全模组...ok"
    else
        echo_red "关闭 SElinux 安全模组...failed"
        ruturn 1
    fi
}


##  Swap 交换分区
close_swap() {
    swapoff -a
    sed -i "s/\/dev\/mapper\/centos-swap/\#\/dev\/mapper\/centos-swap/g" /etc/fstab

    memory_info=$(free -m)
    if echo "$memory_info" | grep -q "Swap:" && echo "$memory_info" | grep -q "0\s*0\s*0"; then
        free -m
        echo_green "关闭 Swap 交换分区...ok"
    else
        echo_red "关闭Swap 交换分区...failed"
        return 1
    fi
}

## 修改 iptable 桥接规则
set_rules() {
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF
modprobe br_netfilter
sysctl -w net.ipv4.ip_forward=1 
sysctl --system
echo_green "修改 iptable 桥接规则...ok"
}


## 加载 IPVS 内核模块
use_ipvs() {
cat > /etc/sysconfig/modules/ipvs.modules <<EOF
#!/bin/bash
modprobe -- ip_vs
modprobe -- ip_vs_rr
modprobe -- ip_vs_wrr
modprobe -- ip_vs_sh
modprobe -- nf_conntrack_ipv4
EOF
chmod 755 /etc/sysconfig/modules/ipvs.modules && bash /etc/sysconfig/modules/ipvs.modules && lsmod | grep -e ip_vs -e nf_conntrack_ipv4
echo_green "加载 IPVS 内核模块...ok"
}


## 安装一些基础工具
install_tools() {
    yum install -y ipset ipvsadm bind-utils net-tools bash-completion git
    echo_green "安装 ipset ipvsadm bind-utils net-tools bash-completion git...ok"
}


## 安装 kubelet kubeadm kubectl
install_kubelet_kubeadm_kubectl() {
## 添加仓库
cat << EOF > /etc/yum.repos.d/kubernetes.repo
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
echo_green "安装 kubelet kubeadm kubectl...ok"
}

## 安装 containerd
install_containerd() {
    ## 设置仓库
    yum install -y yum-utils
    yum-config-manager --add-repo \
    https://download.docker.com/linux/centos/docker-ce.repo

    ## 安装
    yum install -y containerd.io
    echo_green "安装 containerd...ok"
}

## 配置 containerd
configure_containerd() {
    ## 写入配置文件
    containerd config default > /etc/containerd/config.toml
    ## 修改配置文件
    sed -i 's/SystemdCgroup = false/SystemdCgroup = true/g' /etc/containerd/config.toml
    sed -i '/^[[:space:]]*\[plugins\."io\.containerd\.grpc\.v1\.cri"\.registry\.mirrors]$/a \
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"] \
          endpoint = ["https://12k5hoeq.mirror.aliyuncs.com"] \
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."k8s.gcr.io"] \
          endpoint = ["https://registry.aliyuncs.com/k8sxio"]' /etc/containerd/config.toml
    ## 确保 CRI 的 sandbox 镜像版本与 k8s 保持一致
    version=$(kubeadm config images list | grep -o 'registry\.k8s\.io/pause:[0-9\.]*' | cut -d ':' -f2)
    origin_sandbox_image=$(cat /etc/containerd/config.toml | grep sandbox_image | sed 's/"/\\"/g' | sed 's/\//\\\//g')
    new_sandbox_image="    sandbox_image = \"registry.cn-hangzhou.aliyuncs.com\/google_containers\/pause:$version\""
    sed -i "s/$origin_sandbox_image/$new_sandbox_image/g" /etc/containerd/config.toml
    echo_green "配置 containerd...ok"
}

## 配置 CRI
configure_cri() {
    crictl config runtime-endpoint /run/containerd/containerd.sock
    systemctl daemon-reload
    echo_green "配置 CRI...ok"
}

## 启动 containerd
start_containerd() {
    systemctl start containerd && systemctl enable containerd
    echo_green "启动 containerd...ok"
}


## 启动 kubelet
start_kukelet() {
    systemctl start kubelet && systemctl enable kubelet
    echo_green "启动 kubelet...ok"
}

## 拉取基础镜像
pull_k8s_images() {
    ## 提前拉取镜像
    kubeadm config images pull \
    --image-repository=registry.cn-hangzhou.aliyuncs.com/google_containers
    echo_green "拉取基础镜像...ok"
}

## 配置 kubeadm
configure_kubeadm() {
    node_name="kube-master"
    if [ -z "$node_name" ]; then
        echo_red "\$1 参数 host_name 不能为空"
        return 1
    fi
    
    kubeadm config print init-defaults > $HOME/.kubengr/kubeadm.yaml

    ip_address=$(ifconfig eth0 | grep -oP '(?<=inet\s)\d+(\.\d+){3}')
    sed -i "s/^\([[:blank:]]*advertiseAddress:[[:blank:]]*\)1.2.3.4\$/\1$ip_address/" $HOME/.kubengr/kubeadm.yaml

    ## 修改节点名称
    sed -i "s/^\([[:blank:]]*name:[[:blank:]]*\)node\$/\1$node_name/" $HOME/.kubengr/kubeadm.yaml


    ## 修改为国内镜像源
    sed -i 's/imageRepository: registry.k8s.io/imageRepository: registry.cn-hangzhou.aliyuncs.com\/google_containers/g' $HOME/.kubengr/kubeadm.yaml

    ## 修改 pod 子网范围，需和网络插件保持一致
    if ! grep -q 'podSubnet:' $HOME/.kubengr/kubeadm.yaml; then
        sed -i '/serviceSubnet:/i\  podSubnet: 10.244.0.0/16' $HOME/.kubengr/kubeadm.yaml
    fi

    ## 配置 KubeProxy 使用 IPVS
    if ! grep -q 'kind: KubeProxyConfiguration' $HOME/.kubengr/kubeadm.yaml; then
        sed -i '$ a\---\napiVersion: kubeproxy.config.k8s.io/v1alpha1\nkind: KubeProxyConfiguration\nmode: ipvs' $HOME/.kubengr/kubeadm.yaml
    fi

    ## 配置 Kubelet cgroupDriver 使用 systemd
    if ! grep -q 'kind: KubeletConfiguration' $HOME/.kubengr/kubeadm.yaml; then
        sed -i '$ a\---\napiVersion: kubelet.config.k8s.io/v1beta1\nkind: KubeletConfiguration\ncgroupDriver: systemd' $HOME/.kubengr/kubeadm.yaml
    fi
    echo_green "配置 kubeadm...ok"
}

## 初始化 Master 节点
kubeadm_init() {
    kubeadm init --config=$HOME/.kubengr/kubeadm.yaml
    echo_green "kubeadm init master...ok"
}

kubeadm_join() {
    master_ip=$1
    token=$2
    discovery_token_ca_cert_hash=$3

    if [ -z "$master_ip" ]; then
        echo_red "\$1 参数 master_ip 不能为空"
        return 1
    fi

    if [ -z "$token" ]; then
        echo_red "\$2 参数 token 不能为空"
        return 1
    fi

    if [ -z "$discovery_token_ca_cert_hash" ]; then
        echo_red "\$3 参数 discovery_token_ca_cert_hash 不能为空"
        return 1
    fi


    kubeadm join $master_ip \
    --token $token \
    --discovery-token-ca-cert-hash $discovery_token_ca_cert_hash
    echo_green "kubeadm init master...ok"
}

## 写入 kubeconfig
write_kubeconfig() {
    mkdir -p $HOME/.kube
    sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
    sudo chown $(id -u):$(id -g) $HOME/.kube/config
    export KUBECONFIG=/etc/kubernetes/admin.conf
    echo_green "写入 kubeconfig...ok"
}

## 安装 helm
install_helm() {
    if command -v helm >/dev/null 2>&1; then
        echo_green "安装 helm...ok"
        return
    fi

    ## 安装前置软件
    yum install -y epel-release 
    yum install -y git jq

    ## 查询最新的版本号
    owner="helm"
    repo="helm"
    response=$(curl -s "https://api.github.com/repos/$owner/$repo/releases")
    latest_version=$(echo "$response" | jq -r '.[0].tag_name')

    ## 下载、解压、安装
    helm_tar="helm-$latest_version-linux-amd64.tar.gz"
    helm_home="/opt/helm"
    mkdir -p $helm_home
    curl -fsSL -o  $helm_home/$helm_tar https://repo.huaweicloud.com/helm/$latest_version/$helm_tar
    tar -zxvf  $helm_home/$helm_tar -C  $helm_home/
    mv  $helm_home/linux-amd64/helm /usr/local/bin/helm

    if command -v helm >/dev/null 2>&1; then
        echo_green "安装 helm...ok"
    else
        echo_red "安装 helm...failed"
        return 1
    fi
}

## 安装 CNI Flannel
install_flannel() {
    # Needs manual creation of namespace to avoid helm error
    kubectl create ns kube-flannel
    kubectl label --overwrite ns kube-flannel pod-security.kubernetes.io/enforce=privileged

    helm repo add flannel https://flannel-io.github.io/flannel/
    helm install flannel --set podCidr="10.244.0.0/16" --namespace kube-flannel flannel/flannel
    echo_green "安装 cni flannel...ok"
}


mkdir -p $HOME/.kubengr

node_type=$1
if [ -z "$node_type" ]; then
    echo_red "\$1 参数 master_ip 不能为空"
    return 1
fi

node_name=$2
if [ -z "$host_name" ]; then
    echo_red "\$1 参数 host_name 不能为空"
    return 1
fi


## 初始化环境
set_aliyun_yum_repo
set_host_name "$node_name"
synchronous_clock
enable_resolution
disable_firewalld
close_selinux
close_swap
set_rules
use_ipvs
install_tools


## 安装 kubelet、kubeadm、kubectl
install_kubelet_kubeadm_kubectl


## 安装配置 containerd CRI
install_containerd
configure_containerd
configure_cri
start_containerd
start_kukelet


## 初始化 Master 节点，安装 CNI 网络插件
if [ "$node_type" == "master" ]; then
    pull_k8s_images
    configure_kubeadm "$node_name"
    kubeadm_init
    write_kubeconfig

    install_helm
    install_flannel
fi


## 初始化 Worker 节点
if [ "$node_type" == "worker" ]; then
    master_ip=$3
    token=$4
    discovery_token_ca_cert_hash=$5
    kubeadm_join "$master_ip" "$token" "$discovery_token_ca_cert_hash"
fi
`
