#!/bin/bash

ZKEVM_AGGLAYER=$INPUT_ZKEVM_AGGLAYER
ZKEVM_BRIDGE_SERVICE=$INPUT_ZKEVM_BRIDGE_SERVICE
ZKEVM_BRIDGE_UI=$INPUT_ZKEVM_BRIDGE_UI
ZKEVM_DAC=$INPUT_ZKEVM_DAC
ZKEVM_NODE=$INPUT_ZKEVM_NODE
BAKE_TIME=$INPUT_BAKE_TIME

# Get Docker
curl -fsSL https://get.docker.com -o install-docker.sh
cat install-docker.sh
sh install-docker.sh --dry-run
sudo sh install-docker.sh

sudo groupadd docker
sudo usermod -aG docker $USER
newgrp docker
sudo chown "$USER":"$USER" /home/"$USER"/.docker -R
sudo chmod g+rwx "$HOME/.docker" -R

# Clone the repository
git clone https://github.com/0xPolygon/agglayer.git
cd agglayer

# Build agglayer if no release tag is given
if [[ $ZKEVM_AGGLAYER =~ ^[0-9a-fA-F]{7}$ ]]; then
    git checkout "$ZKEVM_AGGLAYER"
    docker compose -f docker/docker-compose.yaml build --no-cache agglayer
else
    echo "Skipping building agglayer as release tag provided: $ZKEVM_AGGLAYER"
fi

# Clone and build zkevm-bridge-service if no release tag is given
cd ..
git clone https://github.com/0xPolygonHermez/zkevm-bridge-service.git
cd zkevm-bridge-service
if [[ $ZKEVM_BRIDGE_SERVICE =~ ^[0-9a-fA-F]{7}$ ]]; then
    git checkout "$ZKEVM_BRIDGE_SERVICE"
    docker build -t zkevm-bridge-service:local -f ./Dockerfile .
else
    echo "Skipping building zkevm-bridge-service as release tag provided: $ZKEVM_BRIDGE_SERVICE"
fi

# Clone and build zkevm-bridge-ui if no release tag is given
cd ..
git clone https://github.com/0xPolygonHermez/zkevm-bridge-ui.git
cd zkevm-bridge-ui
if [[ $ZKEVM_BRIDGE_UI =~ ^[0-9a-fA-F]{7}$ ]]; then
    git checkout "$ZKEVM_BRIDGE_UI"
    docker build -t zkevm-bridge-ui:local -f ./Dockerfile .
else
    echo "Skipping building zkevm-bridge-ui as release tag provided: $ZKEVM_BRIDGE_UI"
fi

# Clone and build cdk-data-availability if no release tag is given
cd ..
git clone https://github.com/0xPolygon/cdk-data-availability.git
cd cdk-data-availability
if [[ $ZKEVM_DAC =~ ^[0-9a-fA-F]{7}$ ]]; then
    git checkout "$ZKEVM_DAC"
    docker build -t cdk-data-availability:local -f ./Dockerfile .
else
    echo "Skipping building cdk-data-availability as release tag provided: $ZKEVM_DAC"
fi

# Clone and build cdk-validium-node if no release tag is given
cd ..
git clone https://github.com/0xPolygon/cdk-validium-node.git
cd cdk-validium-node
if [[ $ZKEVM_NODE =~ ^[0-9a-fA-F]{7}$ ]]; then
    git checkout "$ZKEVM_NODE"
    docker build -t cdk-validium-node:local -f ./Dockerfile .
else
    echo "Skipping building cdk-validium-node as release tag provided: $ZKEVM_NODE"
fi

# Get Rust and cargo
curl https://sh.rustup.rs -sSf | bash -s -- -y
echo 'source $HOME/.cargo/env' >> $HOME/.bashrc

# Install Foundry
cd ..
git clone https://github.com/foundry-rs/foundry.git
cd foundry
cargo install --path ./crates/cast --profile local --force --locked

# Clone internal kurtosis-cdk repo
cd ..
git clone https://github.com/0xPolygon/kurtosis-cdk.git
cd kurtosis-cdk

# Install kurtosis
echo "deb [trusted=yes] https://apt.fury.io/kurtosis-tech/ /" | sudo tee /etc/apt/sources.list.d/kurtosis.list
sudo apt update
sudo apt install kurtosis-cli
kurtosis analytics disable

# Install yq
YQ_VERSION=v4.2.0
YQ_BINARY=yq_linux_amd64
curl -LJO https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/${YQ_BINARY}
chmod +x /usr/bin/yq

# Update kurtosis params.yml with custom devnet containers
if [[ $ZKEVM_AGGLAYER =~ ^[0-9a-fA-F]{7}$ ]]; then
    agglayer_tag="local"
    agglayer_docker_hub="agglayer"
else
    agglayer_tag="$ZKEVM_AGGLAYER"
    agglayer_docker_hub="0xpolygon/agglayer"
fi

if [[ $ZKEVM_BRIDGE_SERVICE =~ ^[0-9a-fA-F]{7}$ ]]; then
    bridge_service_tag="local"
    bridge_service_docker_hub="zkevm-bridge-service"
else
    bridge_service_tag="$ZKEVM_BRIDGE_SERVICE"
    bridge_service_docker_hub="hermeznetwork/zkevm-bridge-service"
fi

if [[ $ZKEVM_BRIDGE_UI =~ ^[0-9a-fA-F]{7}$ ]]; then
    bridge_ui_tag="local"
    bridge_ui_docker_hub="zkevm-bridge-ui"
else
    bridge_ui_tag="$ZKEVM_BRIDGE_UI"
    bridge_ui_docker_hub="hermeznetwork/zkevm-bridge-ui"
fi

if [[ $ZKEVM_DAC =~ ^[0-9a-fA-F]{7}$ ]]; then
    dac_tag="local"
    dac_docker_hub="cdk-data-availability"
else
    dac_tag="$ZKEVM_DAC"
    dac_docker_hub="0xpolygon/cdk-data-availability"
fi

if [[ $ZKEVM_NODE =~ ^[0-9a-fA-F]{7}$ ]]; then
    node_tag="local"
    node_docker_hub="cdk-validium-node"
else
    node_tag="$ZKEVM_NODE"
    node_docker_hub="0xpolygon/cdk-validium-node"
fi

yq -Y --in-place ".args.zkevm_agglayer_image = \"$agglayer_docker_hub:$agglayer_tag\"" params.yml
yq -Y --in-place ".args.zkevm_bridge_service_image = \"$bridge_service_docker_hub:$bridge_service_tag\"" params.yml
yq -Y --in-place ".args.zkevm_bridge_ui_image = \"$bridge_ui_docker_hub:$bridge_ui_tag\"" params.yml
yq -Y --in-place ".args.zkevm_da_image = \"$dac_docker_hub:$dac_tag\"" params.yml
yq -Y --in-place ".args.zkevm_node_image = \"$node_docker_hub:$node_tag\"" params.yml

cat params.yml

# curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
#         && sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl 
# mkdir -p ~/.kube && touch ~/.kube/config
# kurtosis gateway &  # Run cmd in background
# sleep 10

# Deploy CDK devnet on local github runner
mkdir -p /opt/kurtosis-engine-logs
OUTPUT_DIRECTORY="/opt/kurtosis-engine-logs"
kurtosis engine logs $OUTPUT_DIRECTORY
kurtosis engine status
# kurtosis clean --all
kurtosis run --enclave cdk-v1 --args-file params.yml --image-download always .
ls /opt/kurtosis-engine-logs

# Monitor and report any potential regressions to CI logs
bake_time="$BAKE_TIME"
end_minute=$(( $(date +'%M') + bake_time))

export ETH_RPC_URL="$(kurtosis port print cdk-v1 zkevm-node-rpc-001 http-rpc)"
INITIAL_STATUS=$(cast rpc zkevm_verifiedBatchNumber 2>/dev/null)
incremented=false

while [ $(date +'%M') -lt $end_minute ]; do
    # Attempt to connect to the service
    if STATUS=$(cast rpc zkevm_verifiedBatchNumber 2>/dev/null); then
        echo "ZKEVM_VERIFIED_BATCH_NUMBER: $STATUS"
        
        # Check if STATUS has incremented
        if [ "$STATUS" != "$INITIAL_STATUS" ]; then
            incremented=true
            echo "ZKEVM_VERIFIED_BATCH_NUMBER successfully incremented to $STATUS. Exiting..."
            exit 0
        fi
    else
        echo "Failed to connect, waiting and retrying..."
        sleep 60
        continue
    fi
    sleep 60
done

if ! $incremented; then
    echo "ZKEVM_VERIFIED_BATCH_NUMBER did not increment. This may indicate chain experienced a regression. Please investigate."
    exit 1
fi

# Install polycli and send transaction load for further integration tests
cd ..
git clone https://github.com/maticnetwork/polygon-cli.git
cd polygon-cli
make install
export PATH="$HOME/go/bin:$PATH"
export PK="0x12d7de8621a77640c9241b2595ba78ce443d05e94090365ab3bb5e19df82c625"
export ETH_RPC_URL="$(kurtosis port print cdk-v1 zkevm-node-rpc-001 http-rpc)"
polycli loadtest --rpc-url "$ETH_RPC_URL" --legacy --private-key "$PK" --verbosity 700 --requests 500 --rate-limit 5 --mode t
polycli loadtest --rpc-url "$ETH_RPC_URL" --legacy --private-key "$PK" --verbosity 700 --requests 500 --rate-limit 10 --mode t
polycli loadtest --rpc-url "$ETH_RPC_URL" --legacy --private-key "$PK" --verbosity 700 --requests 500 --rate-limit 10 --mode 2
