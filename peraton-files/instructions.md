INSTRUCTIONS
------------

all are run from the terminal, bash shell

## Git Repo

https://github.com/sebastian4/grafana

## Installing

# install make
sudo apt install make

# install golang
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt update
sudo apt install golang-go

# install node
curl -sL https://deb.nodesource.com/setup_16.x -o /tmp/nodesource_setup.sh
sudo bash /tmp/nodesource_setup.sh
sudo apt install nodejs
node -v

# install yarn
sudo apt remove cmdtest
sudo apt remove yarn
curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
sudo apt update
sudo apt install yarn

## Getting Repo

cd /home/linus/downloads
mkdir grafanarepomine14

cd /home/linus/downloads/grafanarepomine14
git clone https://github.com/sebastian4/grafana.git

## Preparing front end

cd /home/linus/downloads/grafanarepomine14/grafana/
yarn install --immutable
yarn start

or run the frontendprep.sh script

## Running application

cd /home/linus/downloads/grafanarepomine14/grafana/
make run

or run the backendrun.sh script
