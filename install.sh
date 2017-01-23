# Temporary install script, installs zifd to ~/zif
# replace with make install when proper config stuff has been written
mkdir -p $HOME/zif/tor

cp bin/zifd $HOME/zif
cp tor/torrc $HOME/zif/tor/torrc
cp run.sh $HOME/zif/run.sh

chmod 700 -R $HOME/zif
chmod +x run.sh
