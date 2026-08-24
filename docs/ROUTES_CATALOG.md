# Route Catalog

Total registered paths: **178** across 114 namespaces.

Status from last full live verification (`scripts/verify-all.sh`): 166 OK, 3 empty (upstream had no items), 9 failed (credential-gated or upstream-blocked).

| Route | Name | Example | Verify |
|---|---|---|---|
| `/163/news/rank` | Netease News Rank | `163/news/rank` | OK(20) |
| `/163/news/rank/:category/:type/:time` | Netease News Rank By Params | `163/news/rank/tech/click/day` | OK(20) |
| `/36kr/information/web_news` | 36Kr Latest News | `36kr/information/web_news` | OK(20) |
| `/36kr/newsflashes` | 36Kr Newsflashes | `36kr/newsflashes` | OK(20) |
| `/aljazeera` | Al Jazeera All News | `aljazeera` | OK(25) |
| `/amo/addon/:slug` | Firefox Add-on Versions | `amo/addon/ublock-origin` | OK(25) |
| `/apnews/latest` | AP News Latest | `apnews/latest` | OK(30) |
| `/appstore/price/:country/:type/:id` | Price Drop Watcher | `appstore/price/us/ios/id1444383602` | OK(1) |
| `/appstore/xianmian` | AppSo Daily Limited-Free | `appstore/xianmian` | OK(20) |
| `/archlinux-news` | Arch Linux News | `archlinux-news` | OK(10) |
| `/arstechnica` | Ars Technica Feed | `arstechnica` | OK(20) |
| `/arxiv/category/:category` | arXiv Category New Papers | `arxiv/category/cs.AI` | OK(20) |
| `/arxiv/search/:keyword` | arXiv Search | `arxiv/search/federated+learning` | OK(20) |
| `/aws-blog` | AWS News Blog | `aws-blog` | OK(20) |
| `/baidu/top` | Baidu Hot Search | `baidu/top` | OK(51) |
| `/baidu/top/:board` | Baidu Hot Search Board | `baidu/top/novel` | OK(30) |
| `/bilibili/hot-search` | Hot Search | `bilibili/hot-search` | OK(50) |
| `/bilibili/popular` | Popular Videos | `bilibili/popular` | OK(20) |
| `/bilibili/ranking` | Ranking | `bilibili/ranking` | OK(50) |
| `/bilibili/ranking/:rid` | Ranking by Zone | `bilibili/ranking/1` | OK(50) |
| `/biorxiv/latest` | bioRxiv Latest Papers | `biorxiv/latest` | OK(30) |
| `/caixin/category/:column/:category` | Caixin Column Category | `caixin/category/finance/regulation` | OK(25) |
| `/caixin/latest` | Caixin Latest | `caixin/latest` | OK(19) |
| `/cloudflare-blog` | Cloudflare Blog | `cloudflare-blog` | OK(20) |
| `/cnbc/rss/:id` | CNBC Full Article RSS | `cnbc/rss/-` | OK(30) |
| `/coindesk` | CoinDesk | `coindesk` | OK(25) |
| `/cointelegraph` | Cointelegraph | `cointelegraph` | OK(30) |
| `/commitstrip` | CommitStrip | `commitstrip` | OK(10) |
| `/coolapk/hot` | Hot List | `coolapk/hot` | OK(18) |
| `/coolapk/hot/:type` | Hot List by Type | `coolapk/hot/jrrm` | OK(18) |
| `/coolapk/toutiao` | Headlines | `coolapk/toutiao` | OK(20) |
| `/coolapk/toutiao/:type` | Headlines by Type | `coolapk/toutiao/digest` | OK(19) |
| `/crates/crate/:crate` | crates.io Crate Versions | `crates/crate/serde` | OK(30) |
| `/css-tricks/articles` | Articles | `css-tricks/articles` | OK(15) |
| `/danbooru/posts` | Danbooru Posts | `danbooru/posts` | OK(20) |
| `/debian-news` | Debian News | `debian-news` | OK(6) |
| `/devto/articles` | DEV Community Articles | `devto/articles` | OK(30) |
| `/devto/tag/:tag` | DEV Community Tag | `devto/tag/python` | OK(30) |
| `/distrowatch` | DistroWatch News | `distrowatch` | OK(10) |
| `/dockerhub/tags/*repo` | Docker Hub Image Tags | `dockerhub/tags/library/nginx` | OK(25) |
| `/douban/movie/playing` | Now Playing Movies | `douban/movie/playing` | OK(55) |
| `/douban/movie/playing/:score` | Now Playing Movies by Score | `douban/movie/playing/8` | OK(14) |
| `/douban/movie/weekly` | Weekly Best Movies | `douban/movie/weekly` | OK(10) |
| `/douyin/hot-search` | Hot Search | `douyin/hot-search` | OK(49) |
| `/dribbble/user/:name` | User Shots | `dribbble/user/google` | OK(48) |
| `/dw/rss/:channel` | DW RSS Feed | `dw/rss/rss-en-all` | OK(130) |
| `/economist/:endpoint` | Economist Category | `economist/latest` | OK(300) |
| `/endoflife/product/:product` | End of Life Product Releases | `endoflife/product/nodejs` | OK(26) |
| `/engadget` | Engadget News | `engadget` | OK(20) |
| `/epicgames/free` | Epic Games Free Games | `epicgames/free` | OK(2) |
| `/fedora-magazine` | Fedora Magazine | `fedora-magazine` | OK(10) |
| `/github-blog` | The GitHub Blog | `github-blog` | OK(10) |
| `/github/commits/:owner/:repo` | GitHub Repository Commits | `github/commits/gin-gonic/gin` | OK(30) |
| `/github/gists/:user` | GitHub User Gists | `github/gists/torvalds` | OK(1) |
| `/github/issues/:owner/:repo` | GitHub Repository Issues | `github/issues/golang/go` | OK(23) |
| `/github/pull/:owner/:repo` | GitHub Repository Pull Requests | `github/pull/gin-gonic/gin` | OK(30) |
| `/github/repos/:owner/:repo` | GitHub Repository Releases | `github/repos/DIYgod/RSSHub` | EMPTY(0) |
| `/github/trending/:language` | GitHub Trending | `github/trending/go` | OK(20) |
| `/github/users/:username/repos` | GitHub User Repositories | `github/users/torvalds/repos` | OK(12) |
| `/gitlab/explore/:sort` | GitLab Explore Projects | `gitlab/explore/last_activity_at` | OK(20) |
| `/gitlab/releases/*project` | GitLab Project Releases | `gitlab/releases/gitlab-org/gitlab-runner` | OK(20) |
| `/goblog` | The Go Blog | `goblog` | OK(10) |
| `/hackage/recent` | Hackage Recent Packages | `hackage/recent` | OK(20) |
| `/hackernews/stories` | Hacker News Top Stories | `hackernews/stories` | OK(30) |
| `/homebrew/formula/:formula` | Homebrew Formula Updates | `homebrew/formula/wget` | OK(20) |
| `/ieee-spectrum/topic/:topic` | IEEE Spectrum Topic | `ieee-spectrum/topic/artificial-intelligence` | OK(30) |
| `/ifeng/news` | Ifeng News Headlines | `ifeng/news` | OK(15) |
| `/infzm/channel/:id` | Infzm Channel | `infzm/channel/2` | OK(10) |
| `/infzm/hot` | Infzm Hot Articles | `infzm/hot` | OK(10) |
| `/ithome` | ITHome News | `ithome` | OK(60) |
| `/itsfoss` | It's FOSS Feed | `itsfoss` | OK(15) |
| `/jin10` | Jin10 Market News | `jin10` | OK(20) |
| `/jin10/category/:id` | Jin10 Category News | `jin10/category/36` | OK(90) |
| `/juejin/category/:category` | Juejin Category | `juejin/category/frontend` | OK(20) |
| `/juejin/pins` | Juejin Pins | `juejin/pins` | OK(20) |
| `/juejin/pins/:type` | Juejin Pins By Type | `juejin/pins/hot` | OK(19) |
| `/juejin/posts/:id` | Juejin User Posts | `juejin/posts/3051900006845944` | OK(10) |
| `/juejin/trending/:category/:type` | Juejin Trending | `juejin/trending/all/weekly` | OK(20) |
| `/kubernetes-blog` | Kubernetes Blog | `kubernetes-blog` | OK(50) |
| `/launches/upcoming` | Upcoming Rocket Launches | `launches/upcoming` | OK(10) |
| `/lemmy/posts/:instance` | Lemmy Posts | `lemmy/posts/lemmy.world` | OK(20) |
| `/letterboxd/watchlist/:username` | User Watchlist | `letterboxd/watchlist/matthew` | OK(28) |
| `/linuxdo/latest` | LINUX DO Latest Topics | `linuxdo/latest` | OK(30) |
| `/linuxdo/top` | LINUX DO Top Topics | `linuxdo/top` | OK(50) |
| `/lobsters/hot` | Lobsters Hot | `lobsters/hot` | OK(25) |
| `/lobsters/newest` | Lobsters Newest | `lobsters/newest` | OK(25) |
| `/lobsters/tag/:tag` | Lobsters Tag | `lobsters/tag/go` | OK(25) |
| `/lwn` | LWN.net Headlines | `lwn` | OK(15) |
| `/mastodon/account/:instance/:id` | Mastodon Account Statuses | `mastodon/account/mastodon.social/1` | OK(19) |
| `/mastodon/timeline/:instance` | Mastodon Public Timeline | `mastodon/timeline/fosstodon.org` | OK(20) |
| `/maven/search/:query` | Maven Central Search | `maven/search/guava` | OK(20) |
| `/medium/feed/:user` | Medium Feed | `medium/feed/zhgchgli` | OK(10) |
| `/medrxiv/latest` | medRxiv Latest Papers | `medrxiv/latest` | OK(30) |
| `/microsoft-devblogs` | Microsoft DevBlogs | `microsoft-devblogs` | OK(10) |
| `/miui/community/user/:uid` | Mi Community User Posts | `miui/community/user/1200057564` | OK(10) |
| `/nasa-apod` | NASA Astronomy Picture of the Day | `nasa-apod` | OK(10) |
| `/nature/journal/:journal` | Nature Journal | `nature/journal/nature` | OK(75) |
| `/nine-to-five-google` | 9to5Google | `nine-to-five-google` | OK(100) |
| `/nine-to-five-linux` | 9to5Linux Feed | `nine-to-five-linux` | OK(10) |
| `/nodejs/blog` | Node.js Blog | `nodejs/blog` | OK(1049) |
| `/nodejs/release` | Node.js Releases | `nodejs/release` | OK(20) |
| `/npm/:package` | npm Package Versions | `npm/react` | OK(20) |
| `/npm/search/:keyword` | npm Search | `npm/search/rss` | OK(20) |
| `/nytimes` | NYT Top Stories | `nytimes` | OK(23) |
| `/nytimes/:category` | NYT Section | `nytimes/technology` | OK(26) |
| `/omgubuntu` | OMG! Ubuntu! Feed | `omgubuntu` | OK(18) |
| `/packagist/package/:vendor/:name` | Packagist Package Versions | `packagist/package/symfony/console` | OK(30) |
| `/people/channel/:site/:category` | People Channel | `people/channel/www/59476` | OK(20) |
| `/people/headlines` | People Headlines | `people/headlines` | OK(20) |
| `/phoronix` | Phoronix News | `phoronix` | OK(32) |
| `/phys` | Phys.org News | `phys` | OK(30) |
| `/plos/journal/:journal` | PLOS Journal | `plos/journal/plosone` | OK(30) |
| `/producthunt/today` | Top Products Launching Today | `producthunt/today` | OK(18) |
| `/pubmed/search/:term` | PubMed Search | `pubmed/search/crispr` | OK(20) |
| `/pypi/package/:package` | PyPI Package Versions | `pypi/package/requests` | OK(20) |
| `/pythoninsider` | Python Insider | `pythoninsider` | OK(25) |
| `/qq/fact` | QQ Fact Check | `qq/fact` | OK(10) |
| `/quantamagazine/archive` | Archive | `quantamagazine/archive` | OK(10) |
| `/reddit/:subreddit` | Reddit Subreddit | `reddit/golang` | OK(25) |
| `/ruanyifeng` | 阮一峰的网络日志 | `ruanyifeng` | OK(3) |
| `/rubygems/gem/:gem` | RubyGems Gem Versions | `rubygems/gem/rails` | OK(30) |
| `/rustblog` | The Rust Blog | `rustblog` | OK(10) |
| `/sciencedaily/top` | ScienceDaily Top News | `sciencedaily/top` | OK(60) |
| `/sina/rollnews` | Sina Rolling News | `sina/rollnews` | OK(30) |
| `/sina/rollnews/:lid` | Sina Rolling News Section | `sina/rollnews/2669` | OK(30) |
| `/smashingmagazine` | Latest Articles | `smashingmagazine` | OK(40) |
| `/smashingmagazine/:category` | Category Articles | `smashingmagazine/react` | OK(10) |
| `/smbc` | Saturday Morning Breakfast Cereal | `smbc` | OK(5) |
| `/solidot` | Solidot News | `solidot` | OK(20) |
| `/spaceflight-news` | Spaceflight News | `spaceflight-news` | OK(20) |
| `/sspai` | sspai Feed | `sspai` | OK(10) |
| `/stackexchange/questions/:site` | Stack Exchange Questions | `stackexchange/questions/stackoverflow` | OK(30) |
| `/steam/news/:appid` | Steam Game News | `steam/news/570` | FAIL(500) |
| `/steam/specials` | Steam Specials | `steam/specials` | OK(25) |
| `/tailscale-blog` | Tailscale Blog | `tailscale-blog` | OK(15) |
| `/techcrunch` | TechCrunch Feed | `techcrunch` | OK(20) |
| `/telegram/channel/:channel` | Telegram Channel | `telegram/channel/durov` | OK(20) |
| `/test/cache` | Cache Test | `test/cache` | OK(1) |
| `/thepaper/channel/:id` | ThePaper Channel | `thepaper/channel/25950` | OK(10) |
| `/theverge` | The Verge Feed | `theverge` | OK(10) |
| `/toutiao/user/token/:token` | Toutiao User Profile | `toutiao/user/token/MS4wLjABAAAA_Q07NxeCa4hDPFoRcdphaZOk2X6C8BApfpTPTMLJswI` | OK(12) |
| `/twitch/live/:login` | Live Status | `twitch/live/riotgames` | EMPTY(0) |
| `/twitch/schedule/:login` | Stream Schedule | `twitch/schedule/northernlion` | OK(1) |
| `/twitch/video/:login` | Channel Videos | `twitch/video/riotgames` | OK(20) |
| `/twitch/video/:login/:filter` | Channel Videos by Type | `twitch/video/riotgames/highlights` | OK(20) |
| `/usgs/all-day` | USGS All Earthquakes Past Day | `usgs/all-day` | OK(100) |
| `/usgs/significant` | USGS Significant Earthquakes | `usgs/significant` | EMPTY(0) |
| `/v2ex/hot` | V2EX Hot Topics | `v2ex/hot` | OK(10) |
| `/v2ex/latest` | V2EX Latest Topics | `v2ex/latest` | OK(42) |
| `/v2ex/node/:name` | V2EX Node Topics | `v2ex/node/python` | OK(10) |
| `/v2ex/topic/:id` | V2EX Topic Replies | `v2ex/topic/1` | OK(186) |
| `/vscode-marketplace/extension/:publisher/:name` | VS Code Extension Versions | `vscode-marketplace/extension/esbenp/prettier-vscode` | OK(30) |
| `/wallhaven/search` | Wallhaven Search | `wallhaven/search?q=nature` | OK(24) |
| `/wallstreetcn/hot` | Wallstreetcn Hot Articles | `wallstreetcn/hot` | OK(10) |
| `/wallstreetcn/hot/:period` | Wallstreetcn Hot Articles By Period | `wallstreetcn/hot/week` | OK(20) |
| `/wallstreetcn/live` | Wallstreetcn Live News | `wallstreetcn/live` | OK(100) |
| `/wallstreetcn/live/:category` | Wallstreetcn Live News By Category | `wallstreetcn/live/a-stock` | OK(100) |
| `/wallstreetcn/news` | Wallstreetcn News | `wallstreetcn/news` | OK(20) |
| `/wallstreetcn/news/:category` | Wallstreetcn News By Category | `wallstreetcn/news/shares` | OK(20) |
| `/weibo/hotsearch` | Hot Search | `weibo/hotsearch` | FAIL(500) |
| `/weibo/user/:uid` | User Timeline | `weibo/user/1195230310` | FAIL(500) |
| `/wikipedia/featured/:date` | Wikipedia Featured Content | `wikipedia/featured/2026-08-23` | OK(5) |
| `/wikipedia/onthisday/:monthday` | Wikipedia On This Day | `wikipedia/onthisday/08-24` | OK(75) |
| `/xkcd/latest` | xkcd Latest Comics | `xkcd/latest` | OK(10) |
| `/youtube/channel/:id` | Channel | `youtube/channel/UCX6OQ3DkcsbYNE6H8uQQuVA` | OK(15) |
| `/youtube/playlist/:id` | Playlist | `youtube/playlist/PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z` | OK(14) |
| `/youtube/user/:username` | Channel With User Handle | `youtube/user/@JFlaMusic` | OK(15) |
| `/zaobao/realtime/:section` | Lianhe Zaobao Realtime | `zaobao/realtime/china` | OK(15) |
| `/zhihu/collection/:id` | 知乎收藏夹 | `zhihu/collection/26444956` | FAIL(500) |
| `/zhihu/daily` | 知乎日报 | `zhihu/daily` | OK(4) |
| `/zhihu/daily/section/:sectionId` | 知乎日报 - 合集 | `zhihu/daily/section/2` | OK(20) |
| `/zhihu/hot` | 知乎热榜 | `zhihu/hot` | OK(30) |
| `/zhihu/people/activities/:id` | 知乎用户动态 | `zhihu/people/activities/diygod` | FAIL(500) |
| `/zhihu/people/answers/:id` | 知乎用户回答 | `zhihu/people/answers/diygod` | FAIL(500) |
| `/zhihu/posts/:usertype/:id` | 知乎用户文章 | `zhihu/posts/people/frederchen` | FAIL(500) |
| `/zhihu/question/:questionId` | 知乎问题 | `zhihu/question/59895982` | FAIL(500) |
| `/zhihu/weekly` | 知乎周刊 | `zhihu/weekly` | OK(15) |
| `/zhihu/zhuanlan/:id` | 知乎专栏 | `zhihu/zhuanlan/googledevelopers` | FAIL(500) |
