wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y apt-transport-https
echo "deb https://artifacts.elastic.co/packages/7.x/apt stable main" | sudo tee /etc/apt/sources.list.d/elastic-7.x.list
sudo apt-get update
sudo apt-get install -y elasticsearch
sudo cat << EOF >> /etc/elasticsearch/elasticsearch.yml
network.host: 0.0.0.0
discovery.type: single-node
EOF
sudo systemctl start elasticsearch