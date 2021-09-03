#!/bin/sh

if ! command go &> /dev/null
then
	echo "You need to have go to compile and install Adam."
	exit
fi

echo 'Compiling...'
go build

echo 'Copying adam to /usr/bin...'
sudo cp adam /usr/bin

echo 'Installing adam manual...'
gzip -c adam.1 > adam.1.gz
sudo cp adam.1.gx /usr/share/man/man1/
