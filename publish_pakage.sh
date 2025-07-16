#!/bin/sh

echo publish with tagged version $1
git push origin master
git tag $1
git push origin $1
