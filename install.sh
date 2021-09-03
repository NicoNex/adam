#!/bin/sh

function install {
	if ! command -v go &> /dev/null
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
	sudo cp adam.1.gz /usr/share/man/man1/
}

function uninstall {
	echo 'Removing /usr/bin/adam...'
	sudo rm /usr/bin/adam

	echo "Removing adam's manual"
	sudo rm /usr/share/man/man1/adam.1.gz
}

function usage {
	cat << EOF
Usage: $0 OPTIONS

This script installs or uninstalls Adam.

OPTIONS:
	-i            Install Adam.
	-r            Uninstall Adam.
	-h, --help    Print this help message.
EOF
}

case $1 in
	-h | --help)
		usage
		;;
	-i)
		install
		;;
	-r)
		uninstall
		;;
	*)
		usage
		;;
esac
