#!/bin/bash
# 关闭 一组 redis
portPre=650

for i in `seq 0 2`;
do
  containId=`docker ps |grep  redis-redisbloom${portPre}${i} |grep ${portPre} | awk '{print $1}'|grep -v grep | head -n 1`
  if [[ $containId != "" ]];then
#    docker stop $containId
    docker stop $containId
    docker rm $containId
  fi

done
echo "finish stop and rm all redisbloom docker container"
exit 0