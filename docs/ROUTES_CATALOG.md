# Route Catalog

Total registered paths: **112** across 81 namespaces.

Status from last full live verification (`scripts/verify-all.sh`):

| Route | Name | Example | Verify |
|---|---|---|---|
| `/amo/addon/:slug` | Firefox Add-on Versions | `amo/addon/ublock-origin` | OK(25) |
| `/archlinux-news` | Arch Linux News | `archlinux-news` | OK(10) |
| `/arstechnica` | Ars Technica Feed | `arstechnica` | OK(20) |
| `/arxiv/category/:category` | arXiv Category New Papers | `arxiv/category/cs.AI` | OK(20) |
| `/arxiv/search/:keyword` | arXiv Search | `arxiv/search/federated+learning` | OK(20) |
| `/aws-blog` | AWS News Blog | `aws-blog` | OK(20) |
| `/biorxiv/latest` | bioRxiv Latest Papers | `biorxiv/latest` | OK(30) |
| `/medrxiv/latest` | medRxiv Latest Papers | `medrxiv/latest` | OK(30) |
| `/bitcoinmagazine` | Bitcoin Magazine | `bitcoinmagazine` | OK(10) |
| `/cloudflare-blog` | Cloudflare Blog | `cloudflare-blog` | OK(20) |
| `/coindesk` | CoinDesk | `coindesk` | OK(25) |
| `/cointelegraph` | Cointelegraph | `cointelegraph` | OK(30) |
| `/commitstrip` | CommitStrip | `commitstrip` | OK(10) |
| `/crates/crate/:crate` | crates.io Crate Versions | `crates/crate/serde` | OK(30) |
| `/danbooru/posts` | Danbooru Posts | `danbooru/posts` | OK(20) |
| `/debian-news` | Debian News | `debian-news` | OK(6) |
| `/devto/articles` | DEV Community Articles | `devto/articles` | OK(30) |
| `/devto/tag/:tag` | DEV Community Tag | `devto/tag/python` | OK(30) |
| `/distrowatch` | DistroWatch News | `distrowatch` | OK(10) |
| `/dockerhub/tags/*repo` | Docker Hub Image Tags | `dockerhub/tags/library/nginx` | OK(25) |
| `/endoflife/product/:product` | End of Life Product Releases | `endoflife/product/nodejs` | OK(26) |
| `/epicgames/free` | Epic Games Free Games | `epicgames/free` | OK(2) |
| `/explosm` | Explosm Comics | `explosm` | OK(5) |
| `/fedora-magazine` | Fedora Magazine | `fedora-magazine` | OK(10) |
| `/github/trending/:language` | GitHub Trending | `github/trending/go` | OK(20) |
| `/github/repos/:owner/:repo` | GitHub Repository Releases | `github/repos/DIYgod/RSSHub` | - |
| `/github/users/:username/repos` | GitHub User Repositories | `github/users/torvalds/repos` | OK(12) |
| `/github/commits/:owner/:repo` | GitHub Repository Commits | `github/commits/gin-gonic/gin` | OK(30) |
| `/github/issues/:owner/:repo` | GitHub Repository Issues | `github/issues/golang/go` | OK(24) |
| `/github/pull/:owner/:repo` | GitHub Repository Pull Requests | `github/pull/gin-gonic/gin` | OK(30) |
| `/github/gists/:user` | GitHub User Gists | `github/gists/torvalds` | OK(1) |
| `/github-blog` | The GitHub Blog | `github-blog` | OK(10) |
| `/gitlab/explore/:sort` | GitLab Explore Projects | `gitlab/explore/last_activity_at` | OK(20) |
| `/gitlab/releases/*project` | GitLab Project Releases | `gitlab/releases/gitlab-org/gitlab-runner` | OK(20) |
| `/goblog` | The Go Blog | `goblog` | OK(10) |
| `/hackage/recent` | Hackage Recent Packages | `hackage/recent` | OK(20) |
| `/hackernews/stories` | Hacker News Top Stories | `hackernews/stories` | OK(30) |
| `/homebrew/formula/:formula` | Homebrew Formula Updates | `homebrew/formula/wget` | OK(20) |
| `/ieee-spectrum/topic/:topic` | IEEE Spectrum Topic | `ieee-spectrum/topic/artificial-intelligence` | OK(30) |
| `/ithome` | ITHome News | `ithome` | OK(60) |
| `/itsfoss` | It's FOSS Feed | `itsfoss` | OK(15) |
| `/kubernetes-blog` | Kubernetes Blog | `kubernetes-blog` | OK(50) |
| `/launches/upcoming` | Upcoming Rocket Launches | `launches/upcoming` | OK(10) |
| `/lemmy/posts/:instance` | Lemmy Posts | `lemmy/posts/lemmy.world` | OK(20) |
| `/linuxdo/latest` | LINUX DO Latest Topics | `linuxdo/latest` | OK(30) |
| `/linuxdo/top` | LINUX DO Top Topics | `linuxdo/top` | OK(50) |
| `/lobsters/hot` | Lobsters Hot | `lobsters/hot` | OK(25) |
| `/lobsters/newest` | Lobsters Newest | `lobsters/newest` | OK(25) |
| `/lobsters/tag/:tag` | Lobsters Tag | `lobsters/tag/go` | OK(25) |
| `/lwn` | LWN.net Headlines | `lwn` | OK(15) |
| `/mastodon/account/:instance/:id` | Mastodon Account Statuses | `mastodon/account/mastodon.social/1` | OK(19) |
| `/mastodon/timeline/:instance` | Mastodon Public Timeline | `mastodon/timeline/fosstodon.org` | OK(20) |
| `/maven/search/:query` | Maven Central Search | `maven/search/guava` | OK(20) |
| `/microsoft-devblogs` | Microsoft DevBlogs | `microsoft-devblogs` | OK(10) |
| `/nasa-apod` | NASA Astronomy Picture of the Day | `nasa-apod` | OK(10) |
| `/nature/journal/:journal` | Nature Journal | `nature/journal/nature` | OK(75) |
| `/nine-to-five-google` | 9to5Google | `nine-to-five-google` | OK(100) |
| `/nine-to-five-linux` | 9to5Linux Feed | `nine-to-five-linux` | OK(10) |
| `/noaa-alerts` | NOAA Active Weather Alerts | `noaa-alerts` | OK(20) |
| `/nodejs/blog` | Node.js Blog | `nodejs/blog` | OK(1049) |
| `/nodejs/release` | Node.js Releases | `nodejs/release` | OK(20) |
| `/npm/:package` | npm Package Versions | `npm/react` | OK(20) |
| `/npm/search/:keyword` | npm Search | `npm/search/rss` | OK(20) |
| `/omgubuntu` | OMG! Ubuntu! Feed | `omgubuntu` | OK(18) |
| `/opensuse-news` | openSUSE News | `opensuse-news` | OK(10) |
| `/packagist/package/:vendor/:name` | Packagist Package Versions | `packagist/package/symfony/console` | OK(30) |
| `/phoronix` | Phoronix News | `phoronix` | OK(32) |
| `/phys` | Phys.org News | `phys` | OK(30) |
| `/plos/journal/:journal` | PLOS Journal | `plos/journal/plosone` | OK(30) |
| `/pubmed/search/:term` | PubMed Search | `pubmed/search/crispr` | OK(20) |
| `/pypi/package/:package` | PyPI Package Versions | `pypi/package/requests` | OK(20) |
| `/pythoninsider` | Python Insider | `pythoninsider` | OK(25) |
| `/reddit/:subreddit` | Reddit Subreddit | `reddit/golang` | OK(25) |
| `/ruanyifeng` | 阮一峰的网络日志 | `ruanyifeng` | OK(3) |
| `/rubygems/gem/:gem` | RubyGems Gem Versions | `rubygems/gem/rails` | OK(30) |
| `/rustblog` | The Rust Blog | `rustblog` | OK(10) |
| `/sciencedaily/top` | ScienceDaily Top News | `sciencedaily/top` | OK(60) |
| `/smbc` | Saturday Morning Breakfast Cereal | `smbc` | OK(5) |
| `/solidot` | Solidot News | `solidot` | OK(18) |
| `/spaceflight-news` | Spaceflight News | `spaceflight-news` | OK(20) |
| `/sspai` | sspai Feed | `sspai` | OK(10) |
| `/stackexchange/questions/:site` | Stack Exchange Questions | `stackexchange/questions/stackoverflow` | OK(30) |
| `/steam/news/:appid` | Steam Game News | `steam/news/570` | FAIL |
| `/steam/specials` | Steam Specials | `steam/specials` | OK(25) |
| `/tailscale-blog` | Tailscale Blog | `tailscale-blog` | OK(15) |
| `/techcrunch` | TechCrunch Feed | `techcrunch` | OK(20) |
| `/techne98/blog` | techne98 - blog | `techne98/blog` | FAIL |
| `/telegram/channel/:channel` | Telegram Channel | `telegram/channel/durov` | OK(20) |
| `/test/cache` | Cache Test | `test/cache` | OK(1) |
| `/theverge` | The Verge Feed | `theverge` | OK(10) |
| `/usgs/significant` | USGS Significant Earthquakes | `usgs/significant` | - |
| `/usgs/all-day` | USGS All Earthquakes Past Day | `usgs/all-day` | OK(100) |
| `/v2ex/hot` | V2EX Hot Topics | `v2ex/hot` | OK(10) |
| `/v2ex/latest` | V2EX Latest Topics | `v2ex/latest` | OK(41) |
| `/v2ex/node/:name` | V2EX Node Topics | `v2ex/node/python` | OK(10) |
| `/v2ex/topic/:id` | V2EX Topic Replies | `v2ex/topic/1` | OK(186) |
| `/vscode-marketplace/extension/:publisher/:name` | VS Code Extension Versions | `vscode-marketplace/extension/esbenp/prettier-vscode` | OK(30) |
| `/wallhaven/search` | Wallhaven Search | `wallhaven/search?q=nature` | OK(24) |
| `/wikipedia/onthisday/:monthday` | Wikipedia On This Day | `wikipedia/onthisday/08-24` | OK(75) |
| `/wikipedia/featured/:date` | Wikipedia Featured Content | `wikipedia/featured/2026-08-23` | OK(5) |
| `/xkcd/latest` | xkcd Latest Comics | `xkcd/latest` | OK(10) |
| `/yandere/post` | yande.re Posts | `yandere/post` | OK(20) |
| `/zhihu/hot` | 知乎热榜 | `zhihu/hot?limit=30` | OK(30) |
| `/zhihu/daily` | 知乎日报 | `zhihu/daily` | OK(4) |
| `/zhihu/daily/section/:sectionId` | 知乎日报 - 合集 | `zhihu/daily/section/2` | OK(20) |
| `/zhihu/zhuanlan/:id` | 知乎专栏 | `zhihu/zhuanlan/googledevelopers` | OK(20)* |
| `/zhihu/question/:questionId` | 知乎问题回答 | `zhihu/question/59895982?sort_by=default` | OK(20)* |
| `/zhihu/people/answers/:id` | 知乎用户回答 | `zhihu/people/answers/diygod` | OK(7)* |
| `/zhihu/posts/:usertype/:id` | 知乎用户文章 | `zhihu/posts/people/frederchen` | OK(15)* |
| `/zhihu/people/activities/:id` | 知乎用户动态 | `zhihu/people/activities/kaifulee` | OK(7)* |
| `/zhihu/collection/:id` | 知乎收藏夹 | `zhihu/collection/26444956?limit=20` | OK(20)* |
| `/zhihu/weekly` | 知乎周刊 | `zhihu/weekly` | OK(15) |

\* 需要环境变量 `ZHIHU_COOKIES`（登录 cookie，至少含 `z_c0`）。未配置时这些路由返回明确的错误提示，其余路由匿名可用。上游 `topic`、`timeline`、`all-collections`、`pins` 因依赖 x-zse-96 请求签名暂未移植（见 `routes/zhihu/routes.go` 注释）。
