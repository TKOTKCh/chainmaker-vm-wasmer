#!/bin/bash
# 根据配置，启动一组redis
portPre=750

for i in `seq 0 2`;
do
  containId=`docker run -d -p ${portPre}${i}:6379 --name redis-redisbloom${portPre}$i redislabs/rebloom:latest`
  if [[ $? -eq 0 ]];then
    continue
  else
    exit 2
  fi
done

echo "finish start all redisbloom docker container"
exit 0

