<p align="center">
  <a href="https://github.com/GuGoOrg/GuGoTik">
    <img src="https://avatars.githubusercontent.com/u/140634467?s=200&v=4" width="200" height="200" alt="GuGoTik">
  </a>
</p>

<div align="center">

# GuGoTik

_✨ 第六届字节跳动青训营进阶班后端实战项目第一名，迷你抖音后端 GuGoTik ✨_  


</div>

<p align="center">
  <a href="https://raw.githubusercontent.com/GuGoOrg/GuGoTik/master/LICENSE">
    <img src="https://img.shields.io/github/license/GuGoOrg/GuGoTik" alt="license">
  </a>
  <a href="https://github.com/GuGoOrg/GuGoTik/releases">
    <img src="https://img.shields.io/github/v/release/GuGoOrg/GuGoTik?color=blueviolet&include_prereleases" alt="release">
  </a>
  <a href="https://github.com/GuGoOrg/GuGoTik/actions">
    <img src="https://github.com/GuGoOrg/GuGoTik/actions/workflows/devcheck.yml/badge.svg" alt="action">
  </a>

<p align="center">
  <a href="https://github.com/GuGoOrg/GuGoTik/releases">下载</a>
  ·
  <a href="https://github.com/GuGoOrg/GuGoTik/blob/main/CONTRIBUTING_CN.md">参与贡献</a>
  ·
  <a href="https://z37kw7eggp.feishu.cn/docx/Y3KCdaFMSoKKNjxPOHAcWMiInZb">文档</a>
</p>

<p align="center">
    <img src="https://api.visitorbadge.io/api/visitors?path=https://github.com/GuGoOrg/GuGoTik&label=visitors&countColor=%231758F0" alter="Hello, GuGoTik !"/>
    <p align= "center">GIVE US A STAR PLEASE MY SIR !!! | 请给我们一个 Star 求求了 !!!</p>
</p>

GuGoTik是 第六届字节跳动青训营后端进阶 实战项目，题目为编写一个小型的抖音后端。  

如果你想了解更多信息，请等待 青训营结束后 ，GuGoTik 会提供完整的项目开发文档，请给本项目一个 Star ~
# 贡献者
项目开发者：这是一群来自五湖四海的 Contributors，来自于 WHU，HNU，NJUPT。
- [EpicMo](https://github.com/liaosunny123)  
- [Maples](https://github.com/Maple-pro)  
- [Attacks](https://github.com/Attack825)  
- [amazing-compass](https://github.com/amazing-compass)  
- [XFFFCCCC](https://github.com/XFFFCCCC)  

特别感谢：  
- [Eric](https://github.com/ExerciseBook)  
- [Huang Yongliang](https://github.com/956237586)  
- [nicognaW](https://github.com/nicognaW)  

以及有事而无法参与项目的小伙伴：  
- [Chuanwise](https://github.com/Chuanwise)  

# 项目结构  
- docker: 基础镜像，为项目的Dockerfile提供基础镜像，或者是为 K8S 技术设施提供基础镜像
- scripts: 构建脚本
- src: 项目源代码
  - constant: 项目常量
  - extra: 外部服务依赖
  - idl: idl文件
  - models: 数据模型
  - rpc: Rpc 代码
  - services: 微服务实例
  - storage: 存储相关
  - utils: 辅助代码
  - web: 网关代码
- test: 项目测试
- 其他单文件：Docker Compose 文件和使用的demo环境变量

# 外部服务依赖
- Redis (Cluster)
- PostgreSQL  
- Consul  
- OpenTelemetry Collector  
- FFMpeg  
- Go  

项目推荐使用以下可观测性基础设施：  
- Jaeger
- Victoria Metrics
- Grafana

Profile 性能分析：  
- Pyroscope 

# 自部署流程  
由 梦想珈 RyzeBot 提供自动推送至K8S集群构建流程。  
PR 至 Dev 分支，经过基于 Action 的 UnitTest + Code Analysis + Lint + BuildCheck 后，可合并至 endymx 分支。
endymx 分支会自动触发 CD，构建镜像并推送，由 RyzeBot 完成向 K8S 的推送，自动部署。

# 配置  
GuGoTik可以自动捕获环境变量，也可以以 .env 文件的方式手动提供，覆盖顺序为：  
.env > 环境变量 > DefaultEnv > EmptyEnv(即默认提供空值，由GuGoTik提供运行时特判)

# 构建
## 基于 Standalone
运行 scripts 文件夹下 build-all 脚本，然后运行 run-all 脚本即可，请选择自己平台支持的脚本。
## 基于 Docker  
```bash
docker pull epicmo/gugotik:latest
```
通过交互式终端进入容器后自行运行 GateWay 文件夹下和 Services 文件夹下程序  
## 基于 Docker-Compose
在项目根目录运行：  
注：相关的账号密码设置在 .env.docker.compose 文件查看  
```bash
docker compose up -d
```
