# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version.
VAGRANTFILE_API_VERSION = '2'

# Our subnet, minus the final octet, for all vagrants.
SUBNET_BASE = '172.16.206.'

$script_all = <<FINIS
sudo apt-get update
sudo apt-get install -y software-properties-common python
FINIS

$script_builder = <<FINIS
sudo apt-add-repository -y ppa:ansible/ansible
sudo apt-get update
sudo apt-get install -y ansible
FINIS

#sudo pip install ansible==

# Add standard provider customizations.
def customize_virtualbox(cfg, name, memory=1024)
  cfg.vm.provider 'virtualbox' do |v|
    v.name = 'nomad-demo-' + name
    v.customize ['modifyvm', :id, '--natdnshostresolver1', 'on']
    v.memory = memory
  end
end

Vagrant.configure(2) do |config|
  config.vm.box = 'bento/ubuntu-16.04' # LTS
  config.ssh.insert_key = false

  # Define the CI machine.
  config.vm.define 'builder', primary: true do |cfg|
    cfg.vm.hostname = 'builder.vagrant-dc1.example.com'
    cfg.vm.network :private_network, :ip => SUBNET_BASE + '10'
    cfg.vm.provision 'shell', inline: $script_all
    cfg.vm.provision 'shell', inline: $script_builder
    customize_virtualbox(cfg, 'builder')
  end

  # Define the resource machines.
  hosts = {
    'node01' => 21,
    'node02' => 22,
    'node03' => 23,
    'node04' => 24,
  }
  hosts.each do |name, octet|
    config.vm.define name do |cfg|
      cfg.vm.hostname = name + '.vagrant-dc1.example.com'
      cfg.vm.network :private_network, :ip => SUBNET_BASE + octet.to_s
      cfg.vm.provision 'shell', inline: $script_all
      customize_virtualbox(cfg, name)
    end
  end
end
