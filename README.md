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
  <a href="https://github.com/GuGoOrg/GuGoTik/releases">Download</a>
  ·
  <a href="https://github.com/GuGoOrg/GuGoTik/blob/main/CONTRIBUTING.md">Contribute</a>
  ·
  <a href="https://z37kw7eggp.feishu.cn/docx/Y3KCdaFMSoKKNjxPOHAcWMiInZb">Document</a>
</p>

<p align="center">
    <img src="https://api.visitorbadge.io/api/visitors?path=https://github.com/GuGoOrg/GuGoTik&label=visitors&countColor=%231758F0" alter="Hello, GuGoTik !"/>
    <p align= "center">GIVE US A STAR PLEASE MY SIR !!! | 请给我们一个 Star 求求了 !!!</p>
</p>

GuGoTik is sixth practice project of ByteDance campus, themed of a micro TikTok backend.  

If you want to learn more information, please wait util the competition finished. GuGoTik will provide the whole develop document, **please give us a star**~

English | [简体中文](docs/README-CN.md)
# Contributor  
Project Developer: This is a group of Contributors from all over the world, from WHU, HNU, NJUPT.  
- [EpicMo](https://github.com/liaosunny123)
- [Maples](https://github.com/Maple-pro)
- [Attacks](https://github.com/Attack825)
- [amazing-compass](https://github.com/amazing-compass)
- [XFFFCCCC](https://github.com/XFFFCCCC)

Special Thanks：
- [Eric](https://github.com/ExerciseBook)
- [Huang Yongliang](https://github.com/956237586)
- [nicognaW](https://github.com/nicognaW)

And the friends who can not participle in our development due to the personal things：
- [Chuanwise](https://github.com/Chuanwise)

# File structure 
- docker: Built as basic image for project's dockerfile or kubernetes's basic infrastructure.  
- scripts: Build scripts
- src: The source of the project  
    - constant: The project constant  
    - extra: The service being relied upon  
    - idl: idl  
    - models: Data model  
    - rpc: Rpc code  
    - services: Micro services instance  
    - storage: About storage  
    - utils: Utils code  
    - web: Gateway code  
- test: Test of the project  
- Others：Docker Compose file and demo env

# Extern service  
- Redis (Cluster)
- PostgreSQL
- Consul
- OpenTelemetry Collector
- FFMpeg
- Go

Recommend：
- Jaeger
- Victoria Metrics
- Grafana

Profile Analysis：
- Pyroscope

# Build process
RyzeBot of 梦想珈 will build image and push to kubernetes  
The PR to Dev branch can be merged into the Master branch after the Action based UnitTest + Code Analysis + Lint + BuildCheck.  
The Master branch will automatically trigger the CD, build the image and push it, and RyzeBot will complete the push to K8S for automatic deployment.  

# Config
GuGoTik can automatically capture environement variables or provide them manually as .env files in the order of coverage:  
.Env > environement variable > DefaultEnv > EmptyEnv (that is, null value is provided by default, and runtime special judgment is provided by GuGoTik)  

# Build
## Standalone
Run the build-all script in the scripts folder, and then run the run-all script. Please select the script supported by your platform.  
## Docker
```bash
docker pull epicmo/gugotik:latest
```
After entering the container through the interactive end point, run the programs under the GateWay folder and the Services folder by yourself  
## Docker-Compose
Run in the project root directory:  
Note: The relevant account password settings are viewed in the .env.docker.compose file  
```bash
docker compose up -d
```
