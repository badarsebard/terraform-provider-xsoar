set -x

wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y apt-transport-https
echo "deb https://artifacts.elastic.co/packages/7.x/apt stable main" | sudo tee /etc/apt/sources.list.d/elastic-7.x.list
sudo apt-get update
sudo apt-get install -y elasticsearch nfs-kernel-server
sudo cat << EOF >> /etc/elasticsearch/elasticsearch.yml
network.host: 0.0.0.0
discovery.type: single-node
EOF
sudo systemctl start elasticsearch
sudo systemctl enable elasticsearch

sudo systemctl start nfs-kernel-server.service
echo '/demisto *(rw,sync,no_subtree_check,no_root_squash)' | sudo tee /etc/exports
sudo mkdir /demisto -p
sudo groupadd demisto -g 2001
sudo useradd -m demisto -u 2001 -g 2001
sudo chown demisto:demisto /demisto
sudo exportfs -a