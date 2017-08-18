# Nomad Demo

Tested with VirtualBox 5.1.26 and Vagrant 1.9.7.

On the host machine:

    vagrant up
    vagrant ssh builder

On the builder VM:

    ansible-playbook -i /vagrant/ansible/hosts.vagrant /vagrant/ansible/play_builder.yml
    ansible-playbook -i /vagrant/ansible/hosts.vagrant /vagrant/ansible/play_world.yml

To make sure Consul and Nomad are healthy, visit the web UI by
forwarding a port on your host machine:

    ssh -N -L 8888:127.0.0.1:8500 -l vagrant -i ~/.vagrant.d/insecure_private_key 172.16.206.21 &
    
Then visit [http://localhost:8888/ui/](http://localhost:8888/ui/). There
should be 3 consul, 18 nomad, and 3 nomad-client services passing.

Deploy the "edge_server" program (on the builder VM):

    make -C /vagrant/go/src/nomad && \
    ansible-playbook -i /vagrant/ansible/hosts.vagrant \
        /vagrant/ansible/play_basic_deploy.yml \
        -e program=edge_server -e job=edge-server

Test at [http://172.16.206.21:25000/info](http://172.16.206.21:25000/info).

Deploy the "demo_server" program:

    make -C /vagrant/go/src/nomad && \
    ansible-playbook -i /vagrant/ansible/hosts.vagrant \
        /vagrant/ansible/play_basic_deploy.yml \
        -e program=demo_server -e job=demo-server

Test again at [http://172.16.206.21:25000/info](http://172.16.206.21/info).
There should be upstream servers now.

## Scenarios

The playbooks in the `ansible/scenarios` directory simulate various
operational scenarios. To test one, SSH into the builder VM and run the
following command:

```
ansible-playbook -i /vagrant/ansible/hosts.vagrant /vagrant/ansible/scenarios/<playbook_name>
```

### Playbooks

* `kill_consul_servers.yml`: Sends a SIGKILL to every consul server process.
* `kill_consul.yml`: Sends a SIGKILL to every consul process (clients and servers).
* `kill_nomad_servers.yml`: Sends a SIGKILL to every nomad server process.
* `kill_nomad.yml`: Sends a SIGKILL to every nomad process (clients and servers).
* `kill_consul_and_nomad.yml`: Sends a SIGKILL to ALL consul and nomad processes.

## Known Issues

**Can't select private network interface in job specs.**

Because job specifications are global (i.e., not tied to any node in
particular), we can't specify a listen IP address other than "0.0.0.0".
There are two problems with this:

1. If the machine has a public and a private network interface, we
   probably want to listen on one or the other (and never both).
2. Nomad uses the first (in alpabetical order) working network interface
   it finds, which in Vagrant+VirtualBox land is never the right one. As
   a result, the registered service address in Consul is not correct.

Current workaround for problem #2 is to ignore the service address
reported by Consul and use the node address instead. Not sure yet how to
deal with problem #1.

**Deployment fails if program version hasn't changed.**

Right now the deployment version is based only on a hash of the
executable file, but deploying just a modified job spec is a valid use
case. Computing the version based on the concatenation of the binary and
the job spec would work sometimes, but not when the spec's base template
is the same and the only change is in its variables.

**System doesn't recover if all consul and nomad servers are killed simultaneously.**

After a complete restart of clusters, consul will not register some
services even though the processes are running. Stopping and and
re-running the relevant nomad job restores the system back to a good
state but unfortunately requires downtime.

`¯\_(ツ)_/¯`

## TODO

* Integrate with Terraform
* Integrate with Vault
* Consul ACL
* Nginx in front of Consul UI
