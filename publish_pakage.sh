#!/bin/sh

echo publish with tagged version $1
git tag $1
git push origin $1
