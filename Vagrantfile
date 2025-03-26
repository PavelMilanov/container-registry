Vagrant.configure("2") do |config|
  config.vm.box = "hashicorp-education/ubuntu-24-04"
  config.vm.box_version = "0.1.0"

  config.vm.synced_folder ".", "/project"

  config.vm.network "forwarded_port", guest: 5050, host: 5050

  config.vm.provision "shell", inline: <<-SHELL
    wget https://go.dev/dl/go1.24.1.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.24.1.linux-amd64.tar.gz
    echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
    source ~/.profile
  SHELL

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "2048"
    vb.cpus = 2
  end
end
