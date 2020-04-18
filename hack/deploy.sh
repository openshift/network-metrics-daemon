#!/bin/sh

echo $IMAGE_TAG
# hack for having the node name variable not expanded in 03_daemonset.yaml
export DOLLAR='$' 

for file in $(ls -v deployments/); do
    echo "INFO - Applying file deployments/$file"
    envsubst < deployments/${file} | ${KUBE_EXEC} apply -f -
done;

