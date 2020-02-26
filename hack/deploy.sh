#!/bin/sh

echo $IMAGE_TAG

for file in $(ls -v deployments/); do
    echo "INFO - Applying file deployments/$file"
    envsubst < deployments/${file} | ${KUBE_EXEC} apply -f -
done;

