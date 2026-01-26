Vagrant.configure("2") do |config|
  config.vm.box = "generic/debian12"
  config.vm.box_version = "4.3.12"

  config.vm.define "registry" do |registry|
  registry.vm.synced_folder "vagrant", "/registry"
  registry.vm.network "forwarded_port", guest: 5050, host: 5050
  registry.vm.provision "shell", inline: <<-SHELL
      curl -fsSL https://get.docker.com -o get-docker.sh
      sudo sh ./get-docker.sh
      sudo usermod -aG docker vagrant
  SHELL
  end

  config.vm.define "s3" do |s3|
  s3.vm.synced_folder "vagrant", "/s3"
  s3.vm.network "forwarded_port", guest: 3900, host: 8090
  s3.vm.network "forwarded_port", guest: 3902, host: 8092
  s3.vm.network "forwarded_port", guest: 3903, host: 8093
  s3.vm.network "forwarded_port", guest: 3904, host: 8094
  s3.vm.network "forwarded_port", guest: 3909, host: 8099
  s3.vm.provision "shell", inline: <<-SHELL
      curl -fsSL https://get.docker.com -o get-docker.sh
      sudo sh ./get-docker.sh
      sudo usermod -aG docker vagrant
  SHELL
  end
end
