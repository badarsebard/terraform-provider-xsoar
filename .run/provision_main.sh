sudo /vagrant/local/installer.sh -- -multi-tenant -y -elasticsearch-url=http://elastic.xsoar.local:9200
sudo cp /vagrant/local/license /tmp/demisto.lic
sudo chown demisto:demisto /tmp/demisto.lic
sudo cp /tmp/demisto.lic /var/lib/demisto/
sudo systemctl restart demisto