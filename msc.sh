#!/bin/bash
make
make swg
rsync -av --delete \
	--exclude '.cursor/' \
	--exclude '.git/'	\
	--exclude '.vscode/'	\
	--exclude 'bak/'	\
	--exclude 'build/'	\
	--exclude 'logs/'	\
/home/jino/go/src/ms-gateway/ /mnt/d/sharp/ms-gateway/
