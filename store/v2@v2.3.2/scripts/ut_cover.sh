#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
function store_ut_cover() {
  #在 store目录下执行
  cd ../
  echo "start store ut ----------"
  go test -coverprofile cover.out ./...
  total=$(go tool cover -func=cover.out | tail -1)
  echo ${total}
  rm cover.out
  coverage=$(echo ${total} | grep -P '\d+\.\d+(?=\%)' -o) #如果macOS 不支持grep -P选项，可以通过brew install grep更新grep
  if [ ! $JENKINS_MYSQL_PWD ]; then
    # 如果测试覆盖率低于N，认为ut执行失败
      (( $(awk "BEGIN {print (${coverage} >= $2)}") )) || (echo "$1 单测覆盖率: ${coverage} 低于 $2%"; exit 1)
  else
    if test -z "$GIT_COMMITTER"; then
        echo "no committer, ignore sql insert"
    fi
  fi
}
#指定 UT覆盖率 不低于 80%
store_ut_cover "store" 80
