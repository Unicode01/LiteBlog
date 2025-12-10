#!/bin/bash

set -e  # 遇到错误立即退出

# 颜色定义（用于更好的日志输出）
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印帮助信息
show_help() {
    cat << 'EOF'
LiteBlog 构建脚本

用法: ./build.sh [选项]

选项:
    platform <target>    指定单个编译目标平台（格式：os/arch，例如 linux/amd64）
    compress            启用静态文件（JS/CSS）压缩
    parallel            启用并行编译（加快构建速度）
    clean               仅清理输出目录
    help                显示此帮助信息

示例:
    ./build.sh                           # 编译所有平台
    ./build.sh platform linux/amd64      # 仅编译 Linux AMD64
    ./build.sh compress                  # 编译所有平台并压缩静态文件
    ./build.sh parallel compress         # 并行编译所有平台并压缩静态文件
    ./build.sh clean                     # 清理输出目录

EOF
    exit 0
}

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
    "windows/arm64"
    "freebsd/amd64"
    "freebsd/386"
)

# 项目名称
app_name="LiteBlog"

# 配置选项
compress_static="false"
parallel_build="false"
clean_only="false"

# 解析命令行参数
while [ $# -gt 0 ]; do
    case "$1" in
        platform)
            if [ -z "$2" ]; then
                echo -e "${RED}错误：'platform' 参数后缺少平台值${NC}" >&2
                exit 1
            fi
            platforms=("$2")
            shift 2
            ;;
        compress)
            compress_static="true"
            shift
            ;;
        parallel)
            parallel_build="true"
            shift
            ;;
        clean)
            clean_only="true"
            shift
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}未知参数: $1${NC}" >&2
            echo "使用 './build.sh help' 查看帮助信息"
            exit 1
            ;;
    esac
done

# 输出目录
output_dir="release"

# 需要打包的资源目录
resource_dirs=("configs" "templates" "public")

# 设置黑名单
# 黑名单中的文件或目录不会被打包到最终的 zip 文件中
blacklist=("public/js/inject.js" "public/css/customizestyle.css")

# 额外参数
args=(-ldflags "-s -w" -gcflags "-B" -tags "nomsgpack netgo osusergo")

# 启用性能分析（默认关闭，可根据需要开启）
PGO="false"

# 检查必要的工具
check_tools() {
    echo -e "${BLUE}[检查] 检查必要工具...${NC}"
    
    # 检查 Go 是否安装
    if ! command -v go &> /dev/null; then
        echo -e "${RED}错误：未找到 Go 编译器，请先安装 Go${NC}" >&2
        exit 1
    fi
    echo -e "${GREEN}✓ Go 版本: $(go version)${NC}"
    
    # 检查 zip 是否安装
    if ! command -v zip &> /dev/null; then
        echo -e "${RED}错误：未找到 zip 命令，请先安装 zip${NC}" >&2
        exit 1
    fi
    echo -e "${GREEN}✓ zip 已安装${NC}"
    
    # 如果需要压缩，检查压缩工具
    if [ "$compress_static" == "true" ]; then
        if ! command -v uglifyjs &> /dev/null; then
            echo -e "${YELLOW}警告：未找到 uglifyjs，JS 文件将不会被压缩${NC}"
            echo -e "${YELLOW}  安装方法: npm install -g uglify-js${NC}"
        else
            echo -e "${GREEN}✓ uglifyjs 已安装${NC}"
        fi
        
        if ! command -v uglifycss &> /dev/null; then
            echo -e "${YELLOW}警告：未找到 uglifycss，CSS 文件将不会被压缩${NC}"
            echo -e "${YELLOW}  安装方法: npm install -g uglifycss${NC}"
        else
            echo -e "${GREEN}✓ uglifycss 已安装${NC}"
        fi
    fi
    echo ""
}

# 清理输出目录
clean_output() {
    if [ -d "$output_dir" ]; then
        echo -e "${BLUE}[清理] 删除旧的输出目录...${NC}"
        rm -rf "$output_dir"
        echo -e "${GREEN}✓ 清理完成${NC}"
    fi
}

# 如果只是清理，执行清理后退出
if [ "$clean_only" == "true" ]; then
    clean_output
    exit 0
fi

# 检查工具
check_tools

# 清理并创建输出目录
clean_output
mkdir -p "$output_dir"

# PGO 性能分析
if [ "$PGO" == "true" ]; then
    echo -e "${BLUE}[PGO] 启用性能引导优化...${NC}"
    go test -cpuprofile=cpu.pprof -bench=. || true
fi

# 编译单个平台的函数
build_platform() {
    local platform=$1
    
    # 分割平台和架构
    IFS="/" read -r os arch <<< "$platform"

    # 设置可执行文件后缀
    local ext=""
    if [ "$os" == "windows" ]; then
        ext=".exe"
    fi

    # 编译
    local output_file="$output_dir/${app_name}${ext}"
    echo -e "${BLUE}[构建] 正在构建 $os/$arch...${NC}"

    if [ "$PGO" == "true" ]; then
        env CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o "$output_file" "${args[@]}" -pgo=cpu.pprof
    else
        env CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o "$output_file" "${args[@]}"
    fi

    # 检查是否成功编译
    if [ $? -ne 0 ]; then
        echo -e "${RED}✗ 构建失败: $os/$arch${NC}" >&2
        return 1
    fi
    echo -e "${GREEN}✓ 编译完成: $os/$arch${NC}"

    # 创建临时目录
    local temp_dir="${output_dir}/temp_${os}_${arch}"
    mkdir -p "$temp_dir"

    # 复制可执行文件和资源目录到临时目录
    echo -e "${BLUE}[打包] 复制资源文件...${NC}"
    cp "$output_file" "$temp_dir/"
    for dir in "${resource_dirs[@]}"; do
        if [ -d "$dir" ]; then
            cp -r "$dir" "$temp_dir/"
        else
            echo -e "${YELLOW}警告：资源目录 $dir 不存在！${NC}"
        fi
    done
    
    # 移除黑名单中的文件或目录
    for file in "${blacklist[@]}"; do
        local target_file="$temp_dir/$file"
        if [ -e "$target_file" ]; then
            rm -f "$target_file"
            echo -e "${BLUE}  - 已移除黑名单文件: $file${NC}"
        fi
    done

    # 压缩静态文件
    if [ "$compress_static" == "true" ]; then
        echo -e "${BLUE}[压缩] 正在压缩静态文件...${NC}"
        
        local js_count=0
        local css_count=0
        
        # 压缩 JavaScript 文件
        if command -v uglifyjs &> /dev/null; then
            while IFS= read -r jsfile; do
                [ -z "$jsfile" ] && continue
                local tempfile="${jsfile}.tmp"
                if uglifyjs "$jsfile" -m -o "$tempfile" 2>/dev/null; then
                    local original_size=$(stat -c%s "$jsfile" 2>/dev/null || stat -f%z "$jsfile" 2>/dev/null || echo 0)
                    local compressed_size=$(stat -c%s "$tempfile" 2>/dev/null || stat -f%z "$tempfile" 2>/dev/null || echo 0)

                    if [ "$compressed_size" -lt "$original_size" ] && [ "$compressed_size" -gt 0 ]; then
                        mv "$tempfile" "$jsfile"
                        echo -e "${GREEN}  ✓ JS: $(basename "$jsfile") ($original_size -> $compressed_size bytes)${NC}"
                        js_count=$((js_count + 1))
                    else
                        rm -f "$tempfile"
                    fi
                else
                    rm -f "$tempfile"
                fi
            done < <(find "$temp_dir/public" -type f -name "*.js" 2>/dev/null || true)
        fi

        # 压缩 CSS 文件
        if command -v uglifycss &> /dev/null; then
            while IFS= read -r cssfile; do
                [ -z "$cssfile" ] && continue
                local tempfile="${cssfile}.tmp"
                if uglifycss "$cssfile" > "$tempfile" 2>/dev/null; then
                    local original_size=$(stat -c%s "$cssfile" 2>/dev/null || stat -f%z "$cssfile" 2>/dev/null || echo 0)
                    local compressed_size=$(stat -c%s "$tempfile" 2>/dev/null || stat -f%z "$tempfile" 2>/dev/null || echo 0)

                    if [ "$compressed_size" -lt "$original_size" ] && [ "$compressed_size" -gt 0 ]; then
                        mv "$tempfile" "$cssfile"
                        echo -e "${GREEN}  ✓ CSS: $(basename "$cssfile") ($original_size -> $compressed_size bytes)${NC}"
                        css_count=$((css_count + 1))
                    else
                        rm -f "$tempfile"
                    fi
                else
                    rm -f "$tempfile"
                fi
            done < <(find "$temp_dir/public" -type f -name "*.css" 2>/dev/null || true)
        fi
        
        echo -e "${GREEN}✓ 压缩完成: $js_count 个 JS 文件, $css_count 个 CSS 文件${NC}"
    fi

    # 打包成 zip 文件（保留目录结构）
    local zip_file="${app_name}_${os}_${arch}.zip"
    echo -e "${BLUE}[打包] 创建压缩包 $zip_file...${NC}"
    
    # 进入临时目录并打包
    (cd "$temp_dir" && zip -qr "../$zip_file" .) || {
        echo -e "${RED}✗ 打包失败: $os/$arch${NC}" >&2
        return 1
    }

    # 清理临时目录和编译文件
    rm -rf "$temp_dir"
    rm -f "$output_file"
    
    echo -e "${GREEN}✓ 完成: $zip_file${NC}"
    echo ""
    
    return 0
}

# 主构建流程
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  LiteBlog 构建脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

if [ "$parallel_build" == "true" ] && [ ${#platforms[@]} -gt 1 ]; then
    # 并行构建
    echo -e "${BLUE}[模式] 并行构建模式 (${#platforms[@]} 个平台)${NC}"
    echo ""
    
    # 用于存储后台进程 PID
    pids=()
    
    for platform in "${platforms[@]}"; do
        build_platform "$platform" &
        pids+=($!)
    done
    
    # 等待所有后台任务完成并检查状态
    failed_count=0
    for pid in "${pids[@]}"; do
        if ! wait $pid; then
            failed_count=$((failed_count + 1))
        fi
    done
    
    if [ $failed_count -gt 0 ]; then
        echo ""
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}  有 $failed_count 个平台构建失败${NC}"
        echo -e "${RED}========================================${NC}"
        exit 1
    fi
else
    # 串行构建
    echo -e "${BLUE}[模式] 串行构建模式 (${#platforms[@]} 个平台)${NC}"
    echo ""
    
    failed_platforms=()
    for platform in "${platforms[@]}"; do
        if ! build_platform "$platform"; then
            failed_platforms+=("$platform")
        fi
    done
    
    # 报告失败的平台
    if [ ${#failed_platforms[@]} -gt 0 ]; then
        echo ""
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}  以下平台构建失败:${NC}"
        for platform in "${failed_platforms[@]}"; do
            echo -e "${RED}  - $platform${NC}"
        done
        echo -e "${RED}========================================${NC}"
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  构建完成！${NC}"
echo -e "${GREEN}  输出目录: $output_dir${NC}"
echo -e "${GREEN}========================================${NC}"
