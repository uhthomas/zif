# Temporary install script, installs zifd to ~/zif
# replace with make install when proper config stuff has been written
mkdir -p $HOME/zif/tor
mkdir -p $HOME/.zif

cp bin/zifd $HOME/zif
cp tor/torrc $HOME/zif/tor/torrc
cp config/zifd.toml $HOME/.zif/zifd.toml

chmod 700 -R $HOME/zif
chmod +x run.sh
