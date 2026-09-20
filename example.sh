#!/bin/bash

# JavCLI 使用示例

# 设置代理和 Cookie（请替换为你自己的值）
export SOCKS5_PROXY="127.0.0.1:6153"
# export JAVDB_COOKIES="your_javdb_cookies_here"

echo "=== 搜索功能 ==="
./javcli search SDJS-345 | jq '.movies[0] | {code, title, rating, date}'

echo ""
echo "=== 有码日榜（前3） ==="
./javcli rankings -p daily -t censored | jq '.movies[0:3] | .[] | {code, title, rating}'

echo ""
echo "=== 无码周榜（前3） ==="
./javcli rankings -p weekly -t uncensored
./javcli rankings -p weekly -t uncensored | jq '.movies[0:3] | .[] | {code, title, rating}'

echo ""
echo "=== FC2月榜（前3） ==="
./javcli rankings -p monthly -t fc2 | jq '.movies[0:3] | .[] | {code, title, rating}'

echo ""
echo "=== 获取番号详情 ==="
./javcli detail SDJS-345 | jq '{code, title, rating, duration, actors: [.actors[].name], magnets: .magnets | length}'

echo ""
echo "=== 查看详情中的磁力链接 ==="
./javcli detail SDJS-345 | jq '.magnets[] | {name, size, is_hd, has_subs}'

echo ""
echo "=== 查看演员信息 ==="
./javcli detail IPX-615

echo ""
echo "=== 查看评论信息 ==="
./javcli reviews EBWH-156 -p 1

echo ""
echo "=== 收藏番号（想看） ==="
./javcli bookmarks 

echo ""
echo "=== 添加想看 / 删除想看（需登录） ==="
# ./javcli bookmarks add -t want_watch SDJS-345
# ./javcli bookmarks remove -t want_watch SDJS-345

echo ""
echo "=== 收藏番号（已看） ==="
./javcli bookmarks -t watched

echo ""
echo "=== 收藏番号（演员） ==="
./javcli bookmarks -t actors
