# GuGoTik  
GuGoTik是 第六届字节跳动青训营后端进阶 实战项目，题目为编写一个小型的抖音后端。
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
PR 至 Dev 分支，经过基于 Action 的 UnitTest + Code Analysis + Lint + BuildCheck 后，可合并至 Master 分支。
Master 分支会自动触发 CD，构建镜像并推送，由 RyzeBot 完成向 K8S 的推送，自动部署。

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