#!/bin/bash

## 重设权限
## Only chown what is not already owned by PUID:PGID. `chown -R` rewrote the inode (ctime) of every
## file on every start (slow on a large library, and on Unraid it writes to every data disk
## and to parity); find only reads the tree and changes the files that need it.
## -h: change a symlink itself, never the file it points to.
fix_owner() {
    find "$1" \( ! -user "${PUID}" -o ! -group "${PGID}" \) -exec chown -h "${PUID}:${PGID}" {} +
}

fix_owner /config
if [[ ${PERMS} == true ]]; then
    echo "已设置 PERMS=true，重设 '/media' 目录权限为 ${PUID}:${PGID} 所有（只修改属主不对的文件；媒体文件多时仍需要遍历一遍，耐心等）..."
    fix_owner /media
fi

## 兼容旧的缓存目录
if [[ -d /app/cache ]]; then
    echo "检测到映射了 '/app/cache'，创建软连接 '/config/cache' -> '/app/cache'（如果原有缓存多，花的时间也多，耐心等）"
    fix_owner /app
    if [[ -L /config/cache && $(readlink -f /config/cache) != /app/cache ]]; then
        rm -rf /config/cache &>/dev/null
    fi
    if [[ ! -e /config/cache ]]; then
        su-exec ${PUID}:${PGID} ln -sf /app/cache /config/cache
    fi
else
    if [[ -L /config/cache ]]; then
        echo "检测到 '/config/cache' 指向了不存在的目录 '/app/cache'，删除之，如想保留缓存，请将旧的 'cache' 目录移动到 '/config' 路径下..."
        rm -rf /config/cache &>/dev/null
    fi
fi

umask ${UMASK:-022}
cd /config
exec su-exec "${PUID}:${PGID}" chinesesubfinder
