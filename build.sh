#!/bin/bash

# 定义要编译的目标平台
platforms=(
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/386"
    "windows/arm"
    "freebsd/amd64"
    "freebsd/386"
    "netbsd/amd64"
    "netbsd/386"
    "openbsd/amd64"
    "openbsd/386"
)

# 项目名称
app_name="LiteBlog"

# 输出目录
output_dir="release"
rm -rf $output_dir

# 需要打包的资源目录
resource_dirs=("configs" "templates" "public")

# 设置黑名单
# 黑名单中的文件或目录不会被打包到最终的 zip 文件中
blacklist=("public/js/inject.js" "public/css/customizestyle.css")

# 额外参数
args=(-ldflags "-s -w" -gcflags "-B" -tags "nomsgpack netgo osusergo")

# 启用性能分析
PGO="false"
if [ "$PGO" == "true" ]; then
    echo "Enabling PGO..."
    go test -cpuprofile=cpu.pprof -bench=.
fi

# 创建输出目录
mkdir -p $output_dir

# 编译和打包
for platform in "${platforms[@]}"; do
    # 分割平台和架构
    IFS="/" read -r os arch <<< "$platform"

    # 设置可执行文件后缀
    ext=""
    if [ "$os" == "windows" ]; then
        ext=".exe"
    fi

    # 编译
    output_file="$output_dir/${app_name}${ext}"
    echo "Building for $os/$arch..."

    if [ "$PGO" == "true" ]; then
        env CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o $output_file "${args[@]}" -pgo=cpu.pprof
    else
        env CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o $output_file "${args[@]}"
    fi

    # 检查是否成功编译
    if [ $? -ne 0 ]; then
        echo "Failed to build for $os/$arch"
        continue
    fi

    # 创建临时目录
    temp_dir="${output_dir}/temp_${os}_${arch}"
    mkdir -p "$temp_dir"

    # 复制可执行文件和资源目录到临时目录
    echo "Copying resources to $temp_dir..."
    cp "$output_file" "$temp_dir/"
    for dir in "${resource_dirs[@]}"; do
        if [ -d "$dir" ]; then
            cp -r "$dir" "$temp_dir/"
        else
            echo "Warning: Resource directory $dir not found!"
        fi
    done
    
    # 移除黑名单中的文件或目录
    for file in "${blacklist[@]}"; do
        if [ -e "$temp_dir/$file" ]; then
            rm "$temp_dir/$file"
        fi
    done

    # 打包成 zip 文件（保留目录结构）
    zip_file="${output_dir}/${app_name}_${os}_${arch}.zip"
    echo "Packaging $zip_file..."
    (cd "$temp_dir" && echo "$temp_dir" && zip -qr "../../$zip_file" .)

    # 清理临时目录
    rm -rf "$temp_dir"

    # 删除原始文件
    rm "$output_file"
done

echo "Build and packaging completed!"