# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|

  config.vm.define "elastic" do |elastic|
    elastic.vm.box = "ubuntu/focal64"
    elastic.vm.hostname = "elastic.xsoar.local"
    elastic.vm.provider "virtualbox" do |v|
      v.memory = 4096
      v.cpus = 2
    end
    elastic.vm.provision :hosts, :sync_hosts => true
    elastic.vm.network "private_network", ip: "192.168.56.9"
    elastic.vm.provision "shell", path: ".run/provision_elastic.sh"
  end

  config.vm.define "main" do |main|
    main.vm.box = "ubuntu/focal64"
    main.vm.hostname = "main.xsoar.local"
    main.vm.provider "virtualbox" do |v|
      v.memory = 4096
      v.cpus = 2
    end
    main.vm.provision :hosts, :sync_hosts => true
    main.vm.network "private_network", ip: "192.168.56.10"
    main.vm.provision "shell", path: ".run/provision_main.sh"
  end

  config.vm.define "host1" do |host|
    host.vm.box = "ubuntu/focal64"
    host.vm.hostname = "host1.xsoar.local"
    host.vm.provider "virtualbox" do |v|
     v.memory = 2048
     v.cpus = 1
    end
    host.vm.provision :hosts, :sync_hosts => true
    host.vm.network "private_network", ip: "192.168.56.11"
    host.vm.provision "shell", path: ".run/provision_host.sh"
  end

  config.vm.define "host2" do |host|
    host.vm.box = "ubuntu/focal64"
    host.vm.hostname = "host2.xsoar.local"
    host.vm.provider "virtualbox" do |v|
      v.memory = 2048
      v.cpus = 1
    end
    host.vm.provision :hosts, :sync_hosts => true
    host.vm.network "private_network", ip: "192.168.56.12"
    host.vm.provision "shell", path: ".run/provision_host.sh"
  end

  config.vm.define "host3" do |host|
    host.vm.box = "ubuntu/focal64"
    host.vm.hostname = "host3.xsoar.local"
    host.vm.provider "virtualbox" do |v|
      v.memory = 2048
      v.cpus = 1
    end
    host.vm.provision :hosts, :sync_hosts => true
    host.vm.network "private_network", ip: "192.168.56.13"
    host.vm.synced_folder ENV['HOME']+"/.terraform.d", "/home/vagrant/.terraform.d"
  end

end
