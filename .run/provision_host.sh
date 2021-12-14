set -x

sudo apt-get update
sudo apt-get -y install ca-certificates curl gnupg lsb-release makeself
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get -y install docker-ce docker-ce-cli containerd.io nfs-common
sudo mkdir -p /var/lib/demisto
sudo mount elastic.xsoar.local:/demisto /var/lib/demisto
echo "elastic.xsoar.local:/demisto /var/lib/demisto nfs rsize=8192,wsize=8192,timeo=14,intr" | sudo tee /etc/fstab
sudo groupadd demisto -g 2001
sudo useradd -m demisto -u 2001 -g 2001
sudo usermod -a -G docker demisto
sudo chown demisto:demisto /var/lib/demisto