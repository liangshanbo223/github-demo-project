#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

cur_dir=$(pwd)

# 检查 root 权限
[[ $EUID -ne 0 ]] && echo -e "${red}致命错误：${plain}请使用 root 权限运行此脚本 \n " && exit 1

# 检查系统并设置 release 变量
if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    release=$ID
elif [[ -f /usr/lib/os-release ]]; then
    source /usr/lib/os-release
    release=$ID
else
    echo "检测系统失败，请联系作者！" >&2
    exit 1
fi
echo "当前系统发行版为：$release"

arch() {
    case "$(uname -m)" in
    x86_64 | x64 | amd64) echo 'amd64' ;;
    i*86 | x86) echo '386' ;;
    armv8* | armv8 | arm64 | aarch64) echo 'arm64' ;;
    armv7* | armv7 | arm) echo 'armv7' ;;
    armv6* | armv6) echo 'armv6' ;;
    armv5* | armv5) echo 'armv5' ;;
    s390x) echo 's390x' ;;
    *) echo -e "${green}不支持的 CPU 架构！${plain}" && rm -f install.sh && exit 1 ;;
    esac
}

echo "架构：$(arch)"

install_base() {
    case "${release}" in
    centos | almalinux | rocky | oracle)
        yum -y update && yum install -y -q wget curl tar tzdata
        ;;
    fedora)
        dnf -y update && dnf install -y -q wget curl tar tzdata
        ;;
    arch | manjaro | parch)
        pacman -Syu && pacman -Syu --noconfirm wget curl tar tzdata
        ;;
    opensuse-tumbleweed)
        zypper refresh && zypper -q install -y wget curl tar timezone
        ;;
    *)
        apt-get update && apt-get install -y -q wget curl tar tzdata
        ;;
    esac
}

config_after_install() {
    echo -e "${yellow}正在迁移... ${plain}"
    /usr/local/s-ui/sui migrate

    echo -e "${yellow}安装/更新完成！出于安全考虑，建议修改面板设置 ${plain}"
    read -p "是否继续修改设置 [y/n]？" config_confirm
    if [[ "${config_confirm}" == "y" || "${config_confirm}" == "Y" ]]; then
        echo -e "请输入${yellow}面板端口${plain}（留空则使用现有/默认值）："
        read config_port
        echo -e "请输入${yellow}面板路径${plain}（留空则使用现有/默认值）："
        read config_path

        # 订阅配置
        echo -e "请输入${yellow}订阅端口${plain}（留空则使用现有/默认值）："
        read config_subPort
        echo -e "请输入${yellow}订阅路径${plain}（留空则使用现有/默认值）："
        read config_subPath

        # 设置配置
        echo -e "${yellow}正在初始化，请稍候...${plain}"
        params=""
        [ -z "$config_port" ] || params="$params -port $config_port"
        [ -z "$config_path" ] || params="$params -path $config_path"
        [ -z "$config_subPort" ] || params="$params -subPort $config_subPort"
        [ -z "$config_subPath" ] || params="$params -subPath $config_subPath"
        /usr/local/s-ui/sui setting ${params}

        read -p "是否修改管理员账号密码 [y/n]？" admin_confirm
        if [[ "${admin_confirm}" == "y" || "${admin_confirm}" == "Y" ]]; then
            # 首个管理员账号密码
            read -p "请设置用户名：" config_account
            read -p "请设置密码：" config_password

            # 设置账号密码
            echo -e "${yellow}正在初始化，请稍候...${plain}"
            /usr/local/s-ui/sui admin -username ${config_account} -password ${config_password}
        else
            echo -e "${yellow}当前管理员账号密码：${plain}"
            /usr/local/s-ui/sui admin -show
        fi
    else
        echo -e "${red}已取消...${plain}"
        if [[ ! -f "/usr/local/s-ui/db/s-ui.db" ]]; then
            local usernameTemp=$(head -c 6 /dev/urandom | base64)
            local passwordTemp=$(head -c 6 /dev/urandom | base64)
            echo -e "这是全新安装，出于安全考虑将生成随机登录信息："
            echo -e "###############################################"
            echo -e "${green}用户名：${usernameTemp}${plain}"
            echo -e "${green}密码：${passwordTemp}${plain}"
            echo -e "###############################################"
            echo -e "${red}如果忘记登录信息，可以输入 ${green}s-ui${red} 打开配置菜单${plain}"
            /usr/local/s-ui/sui admin -username ${usernameTemp} -password ${passwordTemp}
        else
            echo -e "${red}这是升级安装，将保留旧设置；如果忘记登录信息，可以输入 ${green}s-ui${red} 打开配置菜单${plain}"
        fi
    fi

    echo -e ""
    read -p "是否安装并配置 Nginx 静态伪装站点（用于 80/443 伪装路由回落） [y/n]？" fake_confirm
    if [[ "${fake_confirm}" == "y" || "${fake_confirm}" == "Y" ]]; then
        install_fake_website
    fi
}

install_fake_website() {
    echo -e "${yellow}开始安装 Nginx 静态伪装站...${plain}"
    # 安装 nginx
    case "${release}" in
        centos | almalinux | rocky | oracle)
            yum install -y epel-release && yum install -y nginx
            ;;
        fedora)
            dnf install -y nginx
            ;;
        arch | manjaro | parch)
            pacman -S --noconfirm nginx
            ;;
        *)
            apt-get update && apt-get install -y nginx
            ;;
    esac

    if ! command -v nginx &>/dev/null; then
        echo -e "${red}Nginx 安装失败，请检查包管理器日志！${plain}"
        return 1
    fi

    echo -e "请选择伪装站模板："
    echo -e "  ${green}1)${plain} 文艺小说站（《经典短篇与古诗词选集》- 动态混淆随机类名）"
    echo -e "  ${green}2)${plain} 极光画廊站（《数字艺术画廊》- 动态几何与自适应 HSL 主题色）"
    echo -e "  ${green}3)${plain} 个人科技博客（《开发者名片》- 极简暗黑毛玻璃风格 & 随机类名颜色）"
    read -p "请输入序号 [1-3，默认 3]: " fake_type
    fake_type=${fake_type:-3}

    read -p "请输入伪装站监听的端口 [默认 80]: " fake_port
    fake_port=${fake_port:-80}

    # 确定根路径
    local web_root="/usr/share/nginx/html"
    if [ ! -d "$web_root" ]; then
        web_root="/var/www/html"
    fi
    mkdir -p "$web_root"

    # 清空旧页面
    rm -rf ${web_root}/*

    # 定义混淆行乱序辅助器
    shuffle_lines() {
        if command -v shuf &>/dev/null; then
            shuf
        else
            awk 'BEGIN{srand();}{print rand()"\t"$0}' | sort -n | cut -f2-
        fi
    }

    # 混淆主处理程序
    obfuscate_fake_website() {
        local file_path="${web_root}/index.html"
        if [ ! -f "$file_path" ]; then
            return 1
        fi

        # 1. 替换类名
        local classes_to_replace=(
            "sui-app-container" "sui-logo" "sui-menu-title" "sui-menu-list" "sui-menu-item"
            "sui-toolbar" "sui-tool-btn" "sui-meta" "sui-content" "sui-gallery-header"
            "sui-tabs" "sui-tab-btn" "sui-gallery-grid" "sui-gallery-card" "sui-gallery-img-container"
            "sui-gallery-info" "sui-gallery-title" "sui-gallery-author" "sui-lightbox"
            "sui-lightbox-content" "sui-lightbox-title" "sui-close-lightbox" "sui-container"
            "sui-hero" "sui-hero-info" "sui-badge" "sui-hero-title" "sui-hero-desc"
            "sui-hero-avatar" "sui-avatar-canvas" "sui-avatar-img" "sui-btn" "sui-btn-primary"
            "sui-section-title" "sui-grid" "sui-card" "sui-card-tag" "sui-card-title"
            "sui-card-desc" "sui-card-link" "sui-tags" "sui-tag"
        )
        for cls in "${classes_to_replace[@]}"; do
            local r_cls="c_$(head -c 16 /dev/urandom | base64 | tr -dc 'a-zA-Z' | head -c 8)"
            sed -i "s/$cls/$r_cls/g" "$file_path"
        done

        # 2. 替换 JS 函数名和变量名
        local js_to_replace=(
            "adjustFont" "switchTheme" "init" "loadArticle" "openPost" "openContact"
            "closeModal" "renderCanvas" "drawAurora" "drawMaze" "drawFractal" "drawMondrian"
            "articles" "fontSize" "postData" "galleryItems" "activePostId"
        )
        for js_fn in "${js_to_replace[@]}"; do
            local r_fn="js_$(head -c 16 /dev/urandom | base64 | tr -dc 'a-zA-Z' | head -c 8)"
            sed -i "s/\<$js_fn\>/$r_fn/g" "$file_path"
        done

        # 3. 替换颜色、圆角、阴影等样式参数
        local r_hue_accent=$((RANDOM % 360))
        local r_hue_primary=$(((r_hue_accent + 40 + RANDOM % 60) % 360))
        local r_hue_secondary=$(((r_hue_primary + 100 + RANDOM % 100) % 360))
        
        local r_color_accent="hsl(${r_hue_accent}, 85%, 55%)"
        local r_color_primary="hsl(${r_hue_primary}, 75%, 60%)"
        local r_color_secondary="hsl(${r_hue_secondary}, 80%, 65%)"
        local r_color_hover="hsl(${r_hue_accent}, 90%, 45%)"
        local r_border_radius="$((RANDOM % 16 + 6))px"
        local r_box_shadow="0 $((RANDOM % 8 + 4))px $((RANDOM % 20 + 10))px rgba(0, 0, 0, 0.$((RANDOM % 8 + 6)))"

        sed -i "s/sui-color-accent/$r_color_accent/g" "$file_path"
        sed -i "s/sui-color-primary/$r_color_primary/g" "$file_path"
        sed -i "s/sui-color-secondary/$r_color_secondary/g" "$file_path"
        sed -i "s/sui-color-hover/$r_color_hover/g" "$file_path"
        sed -i "s/sui-border-radius/$r_border_radius/g" "$file_path"
        sed -i "s/sui-box-shadow/$r_box_shadow/g" "$file_path"

        # 4. 打乱 CSS 声明顺序
        local temp_file="${file_path}.tmp"
        local in_shuffle=0
        local shuffle_block=""
        
        while IFS= read -r line || [[ -n "$line" ]]; do
            if [[ "$line" =~ "/* SHUFFLE */" ]]; then
                in_shuffle=1
                echo "$line" >> "$temp_file"
                shuffle_block=""
            elif [[ "$line" =~ "/* SHUFFLE_END */" ]]; then
                in_shuffle=0
                echo -e "$shuffle_block" | shuffle_lines >> "$temp_file"
                echo "$line" >> "$temp_file"
            elif [ $in_shuffle -eq 1 ]; then
                if [ -z "$shuffle_block" ]; then
                    shuffle_block="$line"
                else
                    shuffle_block="${shuffle_block}\n${line}"
                fi
            else
                echo "$line" >> "$temp_file"
            fi
        done < "$file_path"
        mv -f "$temp_file" "$file_path"

        # 5. 注入隐藏的噪音 DOM 元素
        local r_noise_container="c_$(head -c 16 /dev/urandom | base64 | tr -dc 'a-zA-Z' | head -c 8)"
        local words=("kubernetes" "docker" "microservices" "distributed" "consensus" "raft" "paxos" "database" "indexing" "concurrency" "goroutines" "channels" "asynchronous" "javascript" "vuejs" "reactjs" "webpack" "vite" "compiler" "parser" "lexer" "interpreter" "networking" "protocol" "tcpip" "http3" "quic" "congestion" "bandwidth" "latency" "throughput" "firewall" "encryption" "decryption" "signatures" "certificates" "security" "penetration" "vulnerability" "exploit" "mitigation" "monitoring" "metrics" "telemetry" "observability" "logging" "tracing" "profiling" "optimizations" "garbage" "memory" "allocation" "buffer" "stream" "filesystem" "operating" "kernel" "hypervisor" "virtualization" "cloud" "serverless" "functions" "lambda" "gateway" "proxy" "balancer" "caching" "redis" "memcached" "postgresql" "mysql" "sqlite" "nosql" "mongodb" "cassandra" "elasticsearch" "kafka" "rabbitmq" "grpc" "protobuf" "graphql" "restful" "authentication" "authorization" "oauth" "jwt" "session" "cookies" "headers" "requests" "responses" "websockets" "webrtc" "canvas" "svg" "webgl" "wasm" "assembly")
        
        local noise_sentence=""
        local num_words=$((RANDOM % 6 + 5))
        for ((i=0; i<num_words; i++)); do
            local idx=$((RANDOM % ${#words[@]}))
            noise_sentence="$noise_sentence ${words[$idx]}"
        done
        
        local noise_dom="<div class=\"$r_noise_container\" style=\"display:none !important;visibility:hidden !important;width:0px;height:0px;overflow:hidden;\">$noise_sentence</div>"
        sed -i "s|</body>|${noise_dom}\n</body>|g" "$file_path"

        # 6. 注入哈希干扰注释
        local hash_comment_start="<!--指纹防扫描校验签名: $(head -c 30 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)-->"
        local hash_comment_end="<!--末尾随机签名防哈希碰撞: $(head -c 30 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)-->"
        
        sed -i "1s|^|${hash_comment_start}\n|" "$file_path"
        echo "$hash_comment_end" >> "$file_path"
    }

    # 根据选择写入文件
    if [ "$fake_type" -eq 1 ]; then
        # 1. 小说网页
        local bookshop_names=("云舒书屋" "松风竹影" "知行墨苑" "听雨轩" "半卷书斋" "阅微草堂" "万卷书阁")
        local shop_name=${bookshop_names[$((RANDOM % ${#bookshop_names[@]}))]}

        local poems_titles=("桃花源记" "爱莲说" "陋室铭" "醉翁亭记" "岳阳楼记" "滕王阁序")
        local poems_authors=("陶渊明" "周敦颐" "刘禹锡" "欧阳修" "范仲淹" "王勃")
        local poems_contents=(
            "<p>晋太元中，武陵人捕鱼为业。缘溪行，忘路之远近。忽逢桃花林，夹岸数百步，中无杂树，芳草鲜美，落英缤纷。渔人甚异之，复前行，欲穷其林。</p><p>林尽水源，便得一山，山有小口，仿佛若有光。便舍船，从口入。初极狭，才通人。复行数十步，豁然开朗。土地平旷，屋舍俨然，有良田、美池、桑竹之属。阡陌交通，鸡犬相闻。其中往来种作，男女衣着，悉如外人。黄发垂髫，并怡然自乐。</p>"
            "<p>水陆草木之花，可爱者甚蕃。晋陶渊明独爱菊。自李唐来，世人甚爱牡丹。予独爱莲之出淤泥而不染，濯清涟而不妖，中通外直，不蔓不枝，香远益清，亭亭净植，可远观而不可亵玩焉。</p><p>予谓菊，花之隐逸者也；牡丹，花之富贵者也；莲，花之君子者也。噫！菊之爱，陶后鲜有闻。莲之爱，同予者何人？牡丹之爱，宜乎众矣！</p>"
            "<p>山不在高，有仙则名。水不在深，有龙则灵。斯是陋室，惟吾德馨。苔痕上阶绿，草色入帘青。谈笑有鸿儒，往来无白丁。可以调素琴，阅金经。无丝竹之乱耳，无案牍之劳形。南阳诸葛庐，西蜀子云亭。孔子云：何陋之有？</p>"
            "<p>环滁皆山也。其西南诸峰，林壑尤美，望之蔚然而深秀者，琅琊也。山行六七里，渐闻水声潺潺而泻出于两峰之间者，酿泉也。峰回路转，有亭翼然临于泉上者，醉翁亭也。作亭者谁？山之僧智仙也。名之者谁？太守自谓也。</p>"
            "<p>庆历四年春，滕子京谪守巴陵郡。越明年，政通人和，百废俱兴。乃重修岳阳楼，增其旧制，刻唐贤今人诗赋于其上。属予作文以记之。</p><p>予观夫巴陵胜状，在洞庭一湖。衔远山，吞长江，浩浩汤汤，横无际涯；朝晖夕阴，气象万千。此则岳阳楼之大观也，前人之述备矣。</p>"
            "<p>豫章故郡，洪都新府。星分翼轸，地接衡庐。襟三江而带五湖，控蛮荆而引瓯越。物华天宝，龙光射牛斗之墟；人杰地灵，徐孺下陈蕃之榻。雄州雾列，俊采星驰。台隍枕闽粤之交，宾主尽东南之美。</p>"
        )

        local indices=(0 1 2 3 4 5)
        for ((i=5; i>0; i--)); do
            local j=$((RANDOM % (i + 1)))
            local tmp=${indices[$i]}
            indices[$i]=${indices[$j]}
            indices[$j]=$tmp
        done

        local articles_json=""
        for k in 0 1 2; do
            local idx=${indices[$k]}
            local title=${poems_titles[$idx]}
            local author=${poems_authors[$idx]}
            local content=${poems_contents[$idx]}
            content=$(echo "$content" | sed 's/"/\\"/g')
            if [ $k -gt 0 ]; then
                articles_json="$articles_json,"
            fi
            articles_json="$articles_json{ title: \"$title\", author: \"$author\", content: \"$content\" }"
        done

        cat << EOF > ${web_root}/index.html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>${shop_name} - 经典文学选读</title>
    <style>
        :root {
            --bg-color: #f7f6f2;
            --text-color: #2c2c2a;
            --sidebar-bg: #eae8e1;
            --sidebar-active: #d6d3c9;
            --card-border: rgba(0,0,0,0.06);
            --accent-color: sui-color-accent;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            background: var(--bg-color);
            color: var(--text-color);
            font-family: "PingFang SC", "Lantinghei SC", "Microsoft YaHei", serif;
            transition: background 0.3s, color 0.3s;
            line-height: 1.8;
        }
        .sui-app-container {
            /* SHUFFLE */
            display: flex;
            min-height: 100vh;
            /* SHUFFLE_END */
        }
        aside {
            /* SHUFFLE */
            width: 280px;
            background: var(--sidebar-bg);
            border-right: 1px solid var(--card-border);
            padding: 20px;
            display: flex;
            flex-direction: column;
            transition: background 0.3s;
            /* SHUFFLE_END */
        }
        main {
            /* SHUFFLE */
            flex: 1;
            padding: 40px 60px;
            max-width: 900px;
            margin: 0 auto;
            min-height: 100vh;
            position: relative;
            /* SHUFFLE_END */
        }
        .sui-logo {
            /* SHUFFLE */
            font-size: 20px;
            font-weight: 700;
            font-family: serif;
            color: var(--accent-color);
            margin-bottom: 30px;
            border-bottom: 2px solid var(--accent-color);
            padding-bottom: 10px;
            /* SHUFFLE_END */
        }
        .sui-menu-title {
            /* SHUFFLE */
            font-size: 12px;
            font-weight: 600;
            color: var(--accent-color);
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 10px;
            /* SHUFFLE_END */
        }
        .sui-menu-list {
            /* SHUFFLE */
            list-style: none;
            margin-bottom: 30px;
            /* SHUFFLE_END */
        }
        .sui-menu-item {
            /* SHUFFLE */
            padding: 10px 15px;
            border-radius: sui-border-radius;
            cursor: pointer;
            font-size: 14px;
            margin-bottom: 4px;
            transition: background 0.2s, color 0.2s;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            /* SHUFFLE_END */
        }
        .sui-menu-item:hover {
            background: var(--sidebar-active);
        }
        .sui-menu-item.active {
            background: var(--accent-color);
            color: #ffffff;
        }
        .sui-toolbar {
            /* SHUFFLE */
            display: flex;
            gap: 10px;
            align-items: center;
            margin-bottom: 40px;
            justify-content: flex-end;
            border-bottom: 1px solid var(--card-border);
            padding-bottom: 15px;
            /* SHUFFLE_END */
        }
        .sui-tool-btn {
            /* SHUFFLE */
            background: var(--sidebar-bg);
            border: 1px solid var(--card-border);
            color: var(--text-color);
            padding: 5px 12px;
            border-radius: sui-border-radius;
            font-size: 13px;
            cursor: pointer;
            transition: background 0.2s;
            /* SHUFFLE_END */
        }
        .sui-tool-btn:hover {
            background: var(--sidebar-active);
        }
        article h1 {
            /* SHUFFLE */
            font-family: serif;
            font-size: 32px;
            margin-bottom: 20px;
            font-weight: 800;
            color: var(--accent-color);
            /* SHUFFLE_END */
        }
        article .sui-meta {
            /* SHUFFLE */
            font-size: 13px;
            color: var(--accent-color);
            opacity: 0.8;
            margin-bottom: 30px;
            /* SHUFFLE_END */
        }
        article .sui-content {
            /* SHUFFLE */
            font-size: 18px;
            text-align: justify;
            /* SHUFFLE_END */
        }
        article .sui-content p {
            margin-bottom: 20px;
            text-indent: 2em;
        }
        @media(max-width: 768px) {
            .sui-app-container { flex-direction: column; }
            aside { width: 100%; min-height: auto; border-right: none; border-bottom: 1px solid var(--card-border); }
            main { padding: 30px 20px; }
        }
    </style>
</head>
<body>
    <div class="sui-app-container">
        <aside>
            <div class="sui-logo">📖 ${shop_name}</div>
            <div class="sui-menu-title">文学目录</div>
            <ul class="sui-menu-list" id="menu">
            </ul>
        </aside>
        <main>
            <div class="sui-toolbar">
                <button class="sui-tool-btn" onclick="adjustFont(1)">字 A+</button>
                <button class="sui-tool-btn" onclick="adjustFont(-1)">字 A-</button>
                <button class="sui-tool-btn" onclick="switchTheme('light')">纸白</button>
                <button class="sui-tool-btn" onclick="switchTheme('green')">竹绿</button>
                <button class="sui-tool-btn" onclick="switchTheme('dark')">乌木</button>
            </div>
            <article>
                <h1 id="artTitle">载入中...</h1>
                <div class="sui-meta" id="artMeta">品读文学经典</div>
                <div class="sui-content" id="artContent">
                </div>
            </article>
        </main>
    </div>

    <script>
        const articles = [
            ${articles_json}
        ];
        let fontSize = 18;
        function init() {
            const menu = document.getElementById('menu');
            menu.innerHTML = articles.map((art, index) => {
                return '<li class="sui-menu-item ' + (index === 0 ? 'active' : '') + '" onclick="loadArticle(' + index + ', this)">' + art.title + '</li>';
            }).join('');
            loadArticle(0);
        }
        function loadArticle(index, element) {
            if (element) {
                document.querySelectorAll('.sui-menu-item').forEach(item => item.classList.remove('active'));
                element.classList.add('active');
            }
            const art = articles[index];
            document.getElementById('artTitle').innerText = art.title;
            document.getElementById('artMeta').innerText = "选自《经典短篇集》 · " + art.author;
            document.getElementById('artContent').innerHTML = art.content;
        }
        function adjustFont(scale) {
            fontSize += scale * 2;
            if (fontSize < 12) fontSize = 12;
            if (fontSize > 30) fontSize = 30;
            document.getElementById('artContent').style.fontSize = fontSize + 'px';
        }
        function switchTheme(theme) {
            const root = document.documentElement;
            if (theme === 'light') {
                root.style.setProperty('--bg-color', '#f7f6f2');
                root.style.setProperty('--text-color', '#2c2c2a');
                root.style.setProperty('--sidebar-bg', '#eae8e1');
                root.style.setProperty('--sidebar-active', '#d6d3c9');
            } else if (theme === 'green') {
                root.style.setProperty('--bg-color', '#eef5eb');
                root.style.setProperty('--text-color', '#2b3a26');
                root.style.setProperty('--sidebar-bg', '#e1ecd9');
                root.style.setProperty('--sidebar-active', '#cce0c1');
            } else if (theme === 'dark') {
                root.style.setProperty('--bg-color', '#1c1c1a');
                root.style.setProperty('--text-color', '#cecdca');
                root.style.setProperty('--sidebar-bg', '#252523');
                root.style.setProperty('--sidebar-active', '#383835');
            }
        }
        window.onload = init;
    </script>
</body>
</html>
EOF
    elif [ "$fake_type" -eq 2 ]; then
        # 2. 画廊网页
        cat << 'EOF' > ${web_root}/index.html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>极光画廊 - 数字生成艺术</title>
    <style>
        :root {
            --bg-color: #080a10;
            --text-color: #e2e8f0;
            --card-bg: rgba(20, 24, 38, 0.7);
            --border-color: rgba(255, 255, 255, 0.08);
            --accent-color: sui-color-accent;
            --primary-color: sui-color-primary;
            --secondary-color: sui-color-secondary;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            background: var(--bg-color);
            color: var(--text-color);
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            min-height: 100vh;
            padding: 40px 20px;
        }
        .sui-container {
            /* SHUFFLE */
            max-width: 1200px;
            margin: 0 auto;
            /* SHUFFLE_END */
        }
        .sui-gallery-header {
            /* SHUFFLE */
            text-align: center;
            margin-bottom: 50px;
            /* SHUFFLE_END */
        }
        .sui-gallery-header h1 {
            /* SHUFFLE */
            font-size: 38px;
            font-weight: 800;
            letter-spacing: -0.5px;
            background: linear-gradient(135deg, var(--accent-color), var(--primary-color));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 15px;
            /* SHUFFLE_END */
        }
        .sui-gallery-header p {
            font-size: 16px;
            color: #94a3b8;
        }
        .sui-tabs {
            /* SHUFFLE */
            display: flex;
            justify-content: center;
            gap: 12px;
            margin-bottom: 40px;
            /* SHUFFLE_END */
        }
        .sui-tab-btn {
            /* SHUFFLE */
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            color: var(--text-color);
            padding: 8px 20px;
            border-radius: 30px;
            cursor: pointer;
            font-size: 14px;
            transition: all 0.3s ease;
            /* SHUFFLE_END */
        }
        .sui-tab-btn:hover {
            border-color: var(--accent-color);
            background: rgba(255, 255, 255, 0.05);
        }
        .sui-tab-btn.active {
            background: var(--accent-color);
            border-color: var(--accent-color);
            color: #000000;
            font-weight: 600;
        }
        .sui-gallery-grid {
            /* SHUFFLE */
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
            gap: 30px;
            /* SHUFFLE_END */
        }
        .sui-gallery-card {
            /* SHUFFLE */
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: sui-border-radius;
            overflow: hidden;
            transition: transform 0.3s ease, box-shadow 0.3s ease;
            cursor: pointer;
            display: flex;
            flex-direction: column;
            /* SHUFFLE_END */
        }
        .sui-gallery-card:hover {
            transform: translateY(-5px);
            box-shadow: sui-box-shadow;
            border-color: rgba(255,255,255,0.15);
        }
        .sui-gallery-img-container {
            /* SHUFFLE */
            position: relative;
            width: 100%;
            height: 220px;
            background: #000000;
            overflow: hidden;
            /* SHUFFLE_END */
        }
        .sui-avatar-canvas {
            display: block;
            width: 100%;
            height: 100%;
        }
        .sui-gallery-info {
            /* SHUFFLE */
            padding: 20px;
            flex-grow: 1;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            /* SHUFFLE_END */
        }
        .sui-gallery-title {
            /* SHUFFLE */
            font-size: 18px;
            font-weight: 700;
            margin-bottom: 8px;
            color: #ffffff;
            /* SHUFFLE_END */
        }
        .sui-gallery-author {
            font-size: 13px;
            color: #64748b;
        }
        /* 弹窗 / 灯箱 */
        .sui-lightbox {
            position: fixed;
            top: 0; left: 0; width: 100%; height: 100%;
            background: rgba(3, 7, 18, 0.95);
            display: none;
            align-items: center;
            justify-content: center;
            z-index: 1000;
            padding: 20px;
        }
        .sui-lightbox.active {
            display: flex;
        }
        .sui-lightbox-content {
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: sui-border-radius;
            max-width: 800px;
            width: 100%;
            padding: 24px;
            position: relative;
        }
        .sui-close-lightbox {
            position: absolute;
            top: 15px;
            right: 15px;
            background: none;
            border: none;
            color: var(--text-color);
            font-size: 24px;
            cursor: pointer;
        }
    </style>
</head>
<body>
    <div class="sui-container">
        <div class="sui-gallery-header">
            <h1>AURORA GALLERY</h1>
            <p>基于 HTML5 Canvas 实时渲染的算法数学生成艺术展</p>
        </div>
        <div class="sui-tabs" id="tabs">
            <button class="sui-tab-btn active" onclick="switchTheme('all', this)">全部</button>
            <button class="sui-tab-btn" onclick="switchTheme('aurora', this)">极光流体</button>
            <button class="sui-tab-btn" onclick="switchTheme('maze', this)">欧氏迷宫</button>
            <button class="sui-tab-btn" onclick="switchTheme('fractal', this)">混沌分形</button>
            <button class="sui-tab-btn" onclick="switchTheme('mondrian', this)">格子蒙德</button>
        </div>
        <div class="sui-gallery-grid" id="grid">
        </div>
    </div>

    <div class="sui-lightbox" id="lightbox">
        <div class="sui-lightbox-content">
            <button class="sui-close-lightbox" onclick="closeModal()">×</button>
            <h3 class="sui-lightbox-title" id="lbTitle" style="font-size: 24px; font-weight:700; margin-bottom: 15px; color:#fff;">作品详情</h3>
            <div style="width: 100%; height: 380px; background:#000; border-radius: 8px; overflow:hidden; margin-bottom: 20px;">
                <canvas id="lbCanvas" style="width:100%; height:100%; display:block;"></canvas>
            </div>
            <p id="lbDesc" style="color: #94a3b8; line-height: 1.6;"></p>
        </div>
    </div>

    <script>
        const galleryItems = [
            { id: 1, title: "欧若拉流体 #801", type: "aurora", author: "Fluid Solver", desc: "利用纳维叶-斯托克斯流体动力学的简易多阶三角函数逼近模型，实时在 Canvas 计算流体轨迹并绘制彩色线条。" },
            { id: 2, title: "莫比乌斯迷宫 #402", type: "maze", author: "Maze Generator", desc: "基于深度优先回溯寻路算法，由计算机实时计算并生成一颗随机生成树网格，并绘制出完美的二维几何迷宫。" },
            { id: 3, title: "洛伦兹混沌分形 #910", type: "fractal", author: "Fractal Engine", desc: "基于确定性混沌动力学方程，通过不断迭代渲染出精细的数学递归图形，展现秩序与混乱的边界。" },
            { id: 4, title: "新塑形蒙德里安 #112", type: "mondrian", author: "Grid Composer", desc: "通过二维空间随机正交分割，并利用互补色比例算法填充红、黄、蓝以及黑色块，向皮特·蒙德里安的几何抽象致敬。" },
            { id: 5, title: "太阳风暴重组 #813", type: "aurora", author: "Fluid Solver", desc: "利用随机频率噪声和正弦波叠加，模拟太阳日冕物质抛射在极地磁场产生的炫目红绿渐变光效。" },
            { id: 6, title: "克拉尼对称共振 #505", type: "maze", author: "Maze Generator", desc: "模拟声波震动金属板上的细砂堆积，使用简谐振动公式对屏幕上的像素点进行数学波动干涉绘制。" }
        ];

        function init() {
            renderGrid(galleryItems);
        }

        function renderGrid(items) {
            const grid = document.getElementById('grid');
            grid.innerHTML = items.map(item => {
                return '<div class="sui-gallery-card" onclick="openPost(' + item.id + ')">' +
                    '<div class="sui-gallery-img-container">' +
                        '<canvas class="sui-avatar-canvas" id="cv-' + item.id + '"></canvas>' +
                    '</div>' +
                    '<div class="sui-gallery-info">' +
                        '<div class="sui-gallery-title">' + item.title + '</div>' +
                        '<div class="sui-gallery-author">By ' + item.author + ' · ' + item.type.toUpperCase() + '</div>' +
                    '</div>' +
                '</div>';
            }).join('');

            items.forEach(item => {
                const cv = document.getElementById('cv-' + item.id);
                if (cv) {
                    cv.width = cv.clientWidth || 320;
                    cv.height = 220;
                    renderCanvas(cv, item.type, item.id);
                }
            });
        }

        function renderCanvas(canvas, type, seed) {
            const ctx = canvas.getContext('2d');
            const w = canvas.width;
            const h = canvas.height;
            ctx.clearRect(0,0,w,h);
            
            let random = function() {
                let x = Math.sin(seed++) * 10000;
                return x - Math.floor(x);
            };

            if (type === 'aurora') {
                drawAurora(ctx, w, h, random);
            } else if (type === 'maze') {
                drawMaze(ctx, w, h, random);
            } else if (type === 'fractal') {
                drawFractal(ctx, w, h, random);
            } else if (type === 'mondrian') {
                drawMondrian(ctx, w, h, random);
            }
        }

        function drawAurora(ctx, w, h, random) {
            ctx.fillStyle = "#030712";
            ctx.fillRect(0,0,w,h);
            const lines = 6;
            for (let i = 0; i < lines; i++) {
                ctx.beginPath();
                ctx.strokeStyle = 'hsl(' + (random() * 360) + ', 80%, 60%)';
                ctx.lineWidth = random() * 4 + 2;
                ctx.globalAlpha = 0.35;
                let startY = h * 0.3 + random() * h * 0.4;
                ctx.moveTo(0, startY);
                for (let x = 0; x < w; x += 10) {
                    let y = startY + Math.sin(x * 0.02 + i) * 20 + Math.cos(x * 0.005 + i * 2) * 40;
                    ctx.lineTo(x, y);
                }
                ctx.stroke();
            }
            ctx.globalAlpha = 1.0;
        }

        function drawMaze(ctx, w, h, random) {
            ctx.fillStyle = "#0f172a";
            ctx.fillRect(0,0,w,h);
            ctx.strokeStyle = "rgba(56, 189, 248, 0.4)";
            ctx.lineWidth = 2;
            const step = 15;
            for (let x = 0; x < w; x += step) {
                for (let y = 0; y < h; y += step) {
                    ctx.beginPath();
                    if (random() > 0.5) {
                        ctx.moveTo(x, y);
                        ctx.lineTo(x + step, y + step);
                    } else {
                        ctx.moveTo(x + step, y);
                        ctx.lineTo(x, y + step);
                    }
                    ctx.stroke();
                }
            }
        }

        function drawFractal(ctx, w, h, random) {
            ctx.fillStyle = "#090d16";
            ctx.fillRect(0,0,w,h);
            ctx.strokeStyle = "rgba(168, 85, 247, 0.5)";
            ctx.lineWidth = 1.5;
            
            function drawBranch(x, y, len, angle, depth) {
                if (depth === 0) return;
                let x2 = x + Math.cos(angle) * len;
                let y2 = y + Math.sin(angle) * len;
                ctx.beginPath();
                ctx.moveTo(x, y);
                ctx.lineTo(x2, y2);
                ctx.stroke();
                
                drawBranch(x2, y2, len * 0.75, angle - 0.4, depth - 1);
                drawBranch(x2, y2, len * 0.75, angle + 0.4, depth - 1);
            }
            drawBranch(w / 2, h - 10, 50, -Math.PI / 2, 7);
        }

        function drawMondrian(ctx, w, h, random) {
            ctx.fillStyle = "#f8fafc";
            ctx.fillRect(0,0,w,h);
            
            const colors = ["#ef4444", "#3b82f6", "#f59e0b", "#1e293b", "#ffffff"];
            const cols = 5;
            const rows = 4;
            const cellW = w / cols;
            const cellH = h / rows;

            for (let c = 0; c < cols; c++) {
                for (let r = 0; r < rows; r++) {
                    let color = colors[Math.floor(random() * colors.length)];
                    if (random() > 0.6) {
                        ctx.fillStyle = color;
                        ctx.fillRect(c * cellW, r * cellH, cellW, cellH);
                    }
                    ctx.strokeStyle = "#000000";
                    ctx.lineWidth = 3;
                    ctx.strokeRect(c * cellW, r * cellH, cellW, cellH);
                }
            }
        }

        function switchTheme(type, btn) {
            document.querySelectorAll('.sui-tab-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            if (type === 'all') {
                renderGrid(galleryItems);
            } else {
                const filtered = galleryItems.filter(item => item.type === type);
                renderGrid(filtered);
            }
        }

        let activePostId = null;
        function openPost(id) {
            const item = galleryItems.find(x => x.id === id);
            if (item) {
                activePostId = id;
                document.getElementById('lbTitle').innerText = item.title;
                document.getElementById('lbDesc').innerText = item.desc;
                const lightbox = document.getElementById('lightbox');
                lightbox.classList.add('active');
                
                const lbCanvas = document.getElementById('lbCanvas');
                lbCanvas.width = lbCanvas.clientWidth || 600;
                lbCanvas.height = 380;
                renderCanvas(lbCanvas, item.type, item.id);
            }
        }

        function closeModal() {
            lightbox.classList.remove('active');
            activePostId = null;
        }

        window.onclick = function(e) {
            const modal = document.getElementById('lightbox');
            if (e.target === modal) {
                closeModal();
            }
        }

        window.onload = init;
    </script>
</body>
</html>
EOF
    else
        # 3. 科技博客网页
        local blog_titles=(
            "基于 Go 的高性能网络代理内核设计"
            "探讨 BBR 与 BBRv2 拥塞控制算法"
            "HTTP/3 与 QUIC 协议在现代代理中的演进"
            "Let's Encrypt 证书自动续签原理及不停机回落方案"
            "浅析 Linux 系统网络指纹识别与 Anti-Fingerprint 防御"
        )
        local blog_summaries=(
            "代理内核常面临多用户连接管理与流量统计高吞吐的挑战。通过 Go 内嵌 sing-box，避免了跨进程通信开销，极大地降低了内存抖动并提升了系统的整体响应效能。"
            "BBR 拥塞算法改变了传统基于丢包的拥塞判断，转而通过测量瓶颈链路带宽和往返传播时间来控制数据发送，在弱网高延迟环境下表现极其亮眼。"
            "HTTP/3 抛弃了 TCP，拥抱基于 UDP 的 QUIC 协议。QUIC 不仅消除了 TCP 的头部阻塞问题，还实现了秒级握手与网络迁移自愈，给现代代理带来了跨时代的跃迁。"
            "如何实现 ACME TLS 自动化证书的静默申请？内嵌式 HTTP-01 验证路由结合 Nginx 动态回落方案，让我们的反向代理服务能够做到 24 小时不停机自动完成更新证书逻辑。"
            "主动探测与机器学习能够提取静态 HTTP/CSS 模版的独有哈希指纹以实施规则阻断。为了应对此类探测，我们需要在站点部署上引入多维随机色彩、类名重置和无规的 CSS 排版乱序技术。"
        )
        local blog_contents=(
            "<p>在设计一个高并发、低延迟的多用户网络代理控制面板时，系统架构的选型至关重要。传统的面板多采用外部执行代理程序，通过子进程守护与信号传递的形式进行控制。然而这种跨进程模型不仅带来了繁琐的文件句柄与父子进程自愈难题，更对高并发下的动态流量统计与长连接即时切断带来了数十毫秒以上的时延。</p><p>本站后端基于 Golang 编写，将 sing-box 面板直接作为依赖类库内嵌至 Go 二进制中。由于身处同一运行内存空间，我们得以绕过传统内核文件日志扫描，利用底层 RoutedConnection 连接包装技术，在 TCP/UDP 读写字节流的瞬间通过 atomic.Int64 原子操作记录流量差值。这种无损零拷贝 of 流量审计机制，配合内存中的 Map 维护当前全部活跃 socket 套接字连接，可对流量超标用户实现微秒级切断。这即是极简、无损与平滑系统设计的典范。</p>"
            "<p>拥塞控制是互联网协议的核心所在。传统基于丢包判断的拥塞控制算法（如 Reno, Cubic）在早期的有线网络下极其适用，然而在如今无线网络普及的弱网与共享链路场景中，信道噪声和多径干扰常常引发随机的物理层丢包。Cubic 算法会将这些与网络饱和无关的丢包误判为链路拥塞，从而盲目减半拥塞窗口，引发带宽吞吐量的灾难性暴跌。</p><p>谷歌在 2016 年提出的 BBR (Bottleneck Bandwidth and RTT) 算法，颠覆了传统的拥塞控制模型。它通过实时测量瓶颈链路的最大带宽和最小往返延迟来建立信道模型，将发送速度维持在网络承载极限。BBR 不需要盲目试探丢包阈值，因此能在发生高达 20% 的恶劣分组丢失率时，依然死死咬住高带宽，确保实时流媒体与网络隧道维持强劲吞吐量。目前 BBRv2 更是融入了对公平性和丢包惩罚的多重微调，成为下一代拥塞控制的标杆。</p>"
            "<p>传输控制协议 TCP 经历了长达数十年的发展，其内部所留下的历史债务严重阻碍了传输效率的提升。TCP 连接的建立首先需要三路握手，如果加上 TLS 加密，又需要额外的两路握手，这意味着首包延迟极其严重。更具毁灭性的是，由于 TCP 的按序递交保证，当且仅当一个分组在网络传输中遭遇丢失时，后续所有已到达缓冲区的数据都将被强制阻塞等待重传，这就是著名的 TCP 头部阻塞难题。</p><p>HTTP/3 引入了 QUIC 协议，它基于 UDP 构建，但在应用层重新实现了高可靠的连接机制。QUIC 实现了一路乃至零路握手即可安全建立数据通信；并在底层采用完全独立的流多路复用设计，使单个信道的丢包绝对不会影响其他并行信道的传输。此外，QUIC 支持 Connection ID（连接标识符）代替传统 IP+Port 的四元组，这意味着客户端在 5G 移动基站与 Wi-Fi 之间切换时，隧道连接能在零毫秒内无感重建，彻底治愈了移动终端断网重连的隐痛。</p>"
            "<p>在复杂的网络环境下，保障传输链路的机密性是绝对的底线。为此我们必须通过 Let\\'s Encrypt 或者是 ZeroSSL 申请合法的 TLS 证书。传统的申请方案大多依赖于 Certbot 在系统上监听 80 端口响应 ACME HTTP-01 的 Challenge 请求。然而，这意味着如果我们的系统正在 80 端口运行 Nginx 或 Web 面板，证书申请时就必须强制停机释放端口，或者配置极其繁琐的反向代理重定向规则。</p><p>在本站设计中，我们采用了内嵌式 HTTP-01 验证与零冲突回落系统。后端控制面板会在路由树中预先注册并劫持 \`/.well-known/acme-challenge/*\` 规则。当证书服务商的探测探针请求到达时，面板直接在内存中生成一次性握手密钥响应；其他非验证请求则由面板内置的反向代理模块平滑回落至 Nginx 承载的伪装静态站。这种零冲突的无感不停机申请，使得运维负担下降为零，真正实现了一键配置、终身无忧的自动化证书托管体验。</p>"
            "<p>网络监管系统的主动探测早已跨越了简单的端口连通性检查阶段。现代审查网络会定期模拟真实浏览器，对全球可疑的主机进行 HTTP 特征抓取。如果某台服务器仅响应代理请求而在 80 端口返回 400 Bad Request、404 Not Found 或是千篇一律的模板化伪装站，其很容易被机器学习分类器在数秒内标红并实施 IP 阻断。</p><p>防御这种探测的有效手段即是多维动态随机化与指纹混淆。本站所采用的伪装系统，不仅在部署时使用 Linux 底层的强随机设备 \`urandom\` 生成完全随机的调色板（HSL 渐变）与阴影弧度，更是对 HTML/JS 的数十个 DOM 类名、全局变量名与交互函数进行了深度混淆替换。同时，为打碎静态指纹，CSS 中的每一条属性定义都通过洗牌算法乱序排列，辅以在 DOM 树中注入完全隐藏的技术词组噪音，并利用哈希干扰注释干扰 MD5。通过这套机制，每台节点服务器输出的网页代码具有独一无二的静态指纹，令任何签名审查引擎束手无策。</p>"
        )

        local indices=(0 1 2 3 4)
        for ((i=4; i>0; i--)); do
            local j=$((RANDOM % (i + 1)))
            local tmp=${indices[$i]}
            indices[$i]=${indices[$j]}
            indices[$j]=$tmp
        done

        local blog_articles_json=""
        for k in 0 1 2; do
            local idx=${indices[$k]}
            local title=${blog_titles[$idx]}
            local summary=${blog_summaries[$idx]}
            local content=${blog_contents[$idx]}
            content=$(echo "$content" | sed "s/'/\\\\'/g")
            summary=$(echo "$summary" | sed "s/'/\\\\'/g")
            if [ $k -gt 0 ]; then
                blog_articles_json="$blog_articles_json,"
            fi
            blog_articles_json="$blog_articles_json{ id: $((k+1)), title: '$title', summary: '$summary', content: '$content', date: '2026-06-0$((k+1))', readTime: '$((RANDOM % 10 + 3)) 分钟阅读' }"
        done

        cat << EOF > ${web_root}/index.html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Sui Developer - 极客技术空间</title>
    <style>
        :root {
            --bg-color: #0b0d13;
            --text-color: #f8fafc;
            --card-bg: rgba(22, 28, 45, 0.45);
            --border-color: rgba(255, 255, 255, 0.08);
            --accent-color: sui-color-accent;
            --primary-color: sui-color-primary;
            --secondary-color: sui-color-secondary;
            --hover-color: sui-color-hover;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            background: var(--bg-color);
            color: var(--text-color);
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            line-height: 1.6;
            min-height: 100vh;
            padding: 50px 20px;
        }
        .sui-container {
            /* SHUFFLE */
            max-width: 900px;
            margin: 0 auto;
            /* SHUFFLE_END */
        }
        .sui-hero {
            /* SHUFFLE */
            display: flex;
            align-items: center;
            gap: 24px;
            padding: 30px;
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: sui-border-radius;
            margin-bottom: 40px;
            backdrop-filter: blur(10px);
            /* SHUFFLE_END */
        }
        .sui-hero-avatar {
            /* SHUFFLE */
            width: 80px;
            height: 80px;
            border-radius: 50%;
            background: linear-gradient(135deg, var(--accent-color), var(--primary-color));
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 32px;
            box-shadow: sui-box-shadow;
            /* SHUFFLE_END */
        }
        .sui-hero-info {
            flex: 1;
        }
        .sui-badge {
            /* SHUFFLE */
            display: inline-block;
            font-size: 11px;
            background: rgba(255, 255, 255, 0.08);
            border: 1px solid var(--border-color);
            color: var(--accent-color);
            padding: 2px 8px;
            border-radius: 12px;
            margin-bottom: 6px;
            text-transform: uppercase;
            font-weight: 600;
            /* SHUFFLE_END */
        }
        .sui-hero-title {
            /* SHUFFLE */
            font-size: 24px;
            font-weight: 800;
            color: #ffffff;
            margin-bottom: 4px;
            /* SHUFFLE_END */
        }
        .sui-hero-desc {
            font-size: 14px;
            color: #94a3b8;
        }
        .sui-section-title {
            /* SHUFFLE */
            font-size: 18px;
            font-weight: 700;
            color: var(--accent-color);
            margin-bottom: 20px;
            padding-left: 10px;
            border-left: 3px solid var(--accent-color);
            /* SHUFFLE_END */
        }
        .sui-grid {
            /* SHUFFLE */
            display: flex;
            flex-direction: column;
            gap: 20px;
            /* SHUFFLE_END */
        }
        .sui-card {
            /* SHUFFLE */
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: sui-border-radius;
            padding: 24px;
            transition: all 0.3s ease;
            cursor: pointer;
            backdrop-filter: blur(10px);
            /* SHUFFLE_END */
        }
        .sui-card:hover {
            border-color: var(--accent-color);
            transform: translateY(-2px);
            box-shadow: sui-box-shadow;
        }
        .sui-card-tag {
            /* SHUFFLE */
            font-size: 12px;
            color: var(--accent-color);
            margin-bottom: 8px;
            font-weight: 600;
            /* SHUFFLE_END */
        }
        .sui-card-title {
            /* SHUFFLE */
            font-size: 20px;
            font-weight: 700;
            color: #ffffff;
            margin-bottom: 10px;
            /* SHUFFLE_END */
        }
        .sui-card-desc {
            /* SHUFFLE */
            font-size: 14px;
            color: #94a3b8;
            margin-bottom: 15px;
            text-align: justify;
            /* SHUFFLE_END */
        }
        .sui-card-link {
            /* SHUFFLE */
            font-size: 13px;
            color: var(--accent-color);
            text-decoration: none;
            display: flex;
            align-items: center;
            gap: 6px;
            font-weight: 600;
            /* SHUFFLE_END */
        }
        .sui-card-link:hover {
            color: var(--hover-color);
        }
        /* 弹窗样式 */
        .sui-lightbox {
            position: fixed;
            top: 0; left: 0; width: 100%; height: 100%;
            background: rgba(3, 7, 18, 0.95);
            display: none;
            align-items: center;
            justify-content: center;
            z-index: 1000;
            padding: 20px;
        }
        .sui-lightbox.active {
            display: flex;
        }
        .sui-lightbox-content {
            background: #0f1322;
            border: 1px solid var(--border-color);
            border-radius: sui-border-radius;
            max-width: 700px;
            width: 100%;
            padding: 30px;
            position: relative;
        }
        .sui-close-lightbox {
            position: absolute;
            top: 15px;
            right: 15px;
            background: none;
            border: none;
            color: var(--text-color);
            font-size: 24px;
            cursor: pointer;
        }
        .sui-btn {
            background: var(--accent-color);
            border: none;
            color: #000;
            padding: 8px 16px;
            border-radius: 20px;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.3s;
        }
        .sui-btn:hover {
            background: var(--hover-color);
        }
        @media(max-width: 600px) {
            .sui-hero { flex-direction: column; text-align: center; }
        }
    </style>
</head>
<body>
    <div class="sui-container">
        <div class="sui-hero">
            <div class="sui-hero-avatar">🛠️</div>
            <div class="sui-hero-info">
                <span class="sui-badge">Software Engineer</span>
                <h2 class="sui-hero-title">Sui Developer</h2>
                <p class="sui-hero-desc">专注分布式系统、高性能反向代理与网络协议栈开发。</p>
            </div>
            <button class="sui-btn" onclick="openContact()">与我联系</button>
        </div>

        <h3 class="sui-section-title">技术博客</h3>
        <div class="sui-grid" id="blogGrid">
        </div>
    </div>

    <div class="sui-lightbox" id="postModal">
        <div class="sui-lightbox-content">
            <button class="sui-close-lightbox" onclick="closeModal()">×</button>
            <div id="modalBody" style="color: #cbd5e1;">
            </div>
        </div>
    </div>

    <script>
        const postData = [
            ${blog_articles_json}
        ];

        function init() {
            const grid = document.getElementById('blogGrid');
            grid.innerHTML = postData.map(post => {
                return '<div class="sui-card" onclick="openPost(' + post.id + ')">' +
                    '<div class="sui-card-tag">' + post.date + ' · ' + post.readTime + '</div>' +
                    '<h4 class="sui-card-title">' + post.title + '</h4>' +
                    '<p class="sui-card-desc">' + post.summary + '</p>' +
                    '<span class="sui-card-link">阅读更多 →</span>' +
                '</div>';
            }).join('');
        }

        function openPost(id) {
            const post = postData.find(x => x.id === id);
            if (post) {
                document.getElementById('modalBody').innerHTML = 
                    '<h3 style="font-size:24px; font-weight:800; color:#fff; margin-bottom:10px;">' + post.title + '</h3>' +
                    '<div style="font-size:12px; color:var(--accent-color); margin-bottom:20px;">' + post.date + ' · ' + post.readTime + '</div>' +
                    '<div style="font-size:16px; line-height:1.7; text-align:justify;">' + post.content + '</div>';
                document.getElementById('postModal').classList.add('active');
            }
        }

        function openContact() {
            document.getElementById('modalBody').innerHTML = 
                '<h3 style="font-size:24px; font-weight:800; color:#fff; margin-bottom:15px;">与我联系</h3>' +
                '<p style="margin-bottom:20px; line-height:1.6;">如果您对分布式高并发网关、代理内核研发或高性能网络架构感兴趣，欢迎与我交流技术细节：</p>' +
                '<div style="display:flex; flex-direction:column; gap:12px;">' +
                    '<div><strong>电子邮箱：</strong> <span style="color:var(--accent-color);">admin@sui-net.dev</span></div>' +
                    '<div><strong>GitHub 地址：</strong> <a href="https://github.com/admin8800" target="_blank" style="color:var(--accent-color); text-decoration:none;">github.com/admin8800</a></div>' +
                    '<div><strong>Telegram 频道：</strong> <span style="color:var(--accent-color);">@sui_network_channel</span></div>' +
                '</div>';
            document.getElementById('postModal').classList.add('active');
        }

        function closeModal() {
            document.getElementById('postModal').classList.remove('active');
        }

        window.onclick = function(e) {
            const modal = document.getElementById('postModal');
            if (e.target == modal) {
                closeModal();
            }
        }
    </script>
</body>
</html>
EOF
    fi

    # 配置 Nginx 端口
    if [ -f "/etc/nginx/nginx.conf" ]; then
        local conf_dir="/etc/nginx/conf.d"
        if [ -d "/etc/nginx/sites-enabled" ]; then
            conf_dir="/etc/nginx/sites-enabled"
            rm -f /etc/nginx/sites-enabled/default
        fi
        mkdir -p "$conf_dir"

        cat << EOF > ${conf_dir}/sui-fake.conf
server {
    listen ${fake_port} default_server;
    listen [::]:${fake_port} default_server;
    server_name _;
    root ${web_root};
    index index.html;

    location / {
        try_files \$uri \$uri/ =404;
        add_header X-Server-Cache "HIT-$(head -c 8 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 8)";
        add_header X-Static-Version "v$(($((RANDOM % 9)) + 1)).$(($((RANDOM % 9)) + 1)).$((RANDOM % 99))";
        add_header X-Cache-Status "EXPIRED";
    }
}
EOF
    fi

    if [ -f "/etc/nginx/nginx.conf" ] && [ "${fake_port}" -eq 80 ]; then
        if grep -q "listen       80 default_server" /etc/nginx/nginx.conf; then
            sed -i 's/listen       80/ # listen 80/g' /etc/nginx/nginx.conf
            sed -i 's/listen       \[::\]:80/ # listen [::]:80/g' /etc/nginx/nginx.conf
        fi
    fi

    systemctl daemon-reload
    systemctl restart nginx
    systemctl enable nginx

    echo -e "${green}Nginx 伪装静态站安装配置完成！监听端口：${fake_port}，访问地址：http://localhost:${fake_port}/${plain}"
}

prepare_services() {
    if [[ -f "/etc/systemd/system/sing-box.service" ]]; then
        echo -e "${yellow}正在停止 sing-box 服务... ${plain}"
        systemctl stop sing-box
        rm -f /usr/local/s-ui/bin/sing-box /usr/local/s-ui/bin/runSingbox.sh /usr/local/s-ui/bin/signal
    fi
    if [[ -e "/usr/local/s-ui/bin" ]]; then
        echo -e "###############################################################"
        echo -e "${green}/usr/local/s-ui/bin${red} 目录已存在！"
        echo -e "请检查其中内容，并在迁移后手动删除 ${plain}"
        echo -e "###############################################################"
    fi
    systemctl daemon-reload
}

install_s-ui() {
    cd /tmp/

    echo -e "开始下载 new_s-ui 最新自定义编译版本..."
    wget -N --no-check-certificate -O /tmp/s-ui-linux-$(arch).tar.gz https://raw.githubusercontent.com/liangshanbo223/github-demo-project/main/bin/s-ui-linux-$(arch).tar.gz
    if [[ $? -ne 0 ]]; then
        echo -e "${red}下载 s-ui 失败，请确认您的服务器可以访问 raw.githubusercontent.com${plain}"
        exit 1
    fi

    local keep_config=1
    if [[ -e /usr/local/s-ui/ ]]; then
        echo -e "${yellow}检测到您的系统上已存在旧版 s-ui。${plain}"
        echo -e "请选择安装方式："
        echo -e "  1. 保留配置安装（升级安装，保留所有用户数据、节点设置及自定义文件）"
        echo -e "  2. 覆盖安装（全新安装，清空所有历史数据、节点设置与自定义文件）"
        read -p "请输入选项 [默认 1]: " install_mode
        install_mode=${install_mode:-1}
        if [[ "${install_mode}" == "2" ]]; then
            echo -e "${red}您选择覆盖安装，将清空旧的数据！${plain}"
            keep_config=0
        else
            echo -e "${green}您选择保留配置安装，将保留旧的数据并进行升级。${plain}"
            keep_config=1
        fi
    fi

    if [[ -e /usr/local/s-ui/ ]]; then
        systemctl stop s-ui
    fi

    # 如果是覆盖安装，则彻底清理旧的目录
    if [[ ${keep_config} -eq 0 && -e /usr/local/s-ui/ ]]; then
        echo -e "${yellow}正在清理旧的 s-ui 安装目录...${plain}"
        rm -rf /usr/local/s-ui/
    fi

    # 如果是保留配置安装，为了防范意外，我们在本地临时备份 db 文件夹
    if [[ ${keep_config} -eq 1 && -d "/usr/local/s-ui/db" ]]; then
        echo -e "${yellow}正在备份现有数据库...${plain}"
        rm -rf /tmp/s-ui-db-backup
        mkdir -p /tmp/s-ui-db-backup
        cp -rf /usr/local/s-ui/db/* /tmp/s-ui-db-backup/
    fi

    tar zxvf s-ui-linux-$(arch).tar.gz
    rm s-ui-linux-$(arch).tar.gz -f

    chmod +x s-ui/sui s-ui/s-ui.sh
    cp s-ui/s-ui.sh /usr/bin/s-ui
    mkdir -p /usr/local/
    cp -rf s-ui /usr/local/
    cp -f s-ui/*.service /etc/systemd/system/
    rm -rf s-ui

    # 如果是保留配置安装，且发现 db 被误清理或需要还原确保完整性，则从备份中恢复
    if [[ ${keep_config} -eq 1 && -d "/tmp/s-ui-db-backup" ]]; then
        if [[ ! -f "/usr/local/s-ui/db/s-ui.db" ]]; then
            echo -e "${yellow}检测到升级后数据库丢失，正在从备份中还原...${plain}"
            mkdir -p /usr/local/s-ui/db
            cp -rf /tmp/s-ui-db-backup/* /usr/local/s-ui/db/
        fi
        rm -rf /tmp/s-ui-db-backup
    fi

    config_after_install
    prepare_services

    systemctl enable s-ui --now

    echo -e "${green}s-ui ${last_version}${plain} 安装完成，现已启动并运行..."
    echo -e "你可以通过以下 URL 访问面板：${green}"
    /usr/local/s-ui/sui uri
    echo -e "${plain}"
    echo -e ""
    s-ui help
}

echo -e "${green}正在执行...${plain}"
install_base
install_s-ui $1
