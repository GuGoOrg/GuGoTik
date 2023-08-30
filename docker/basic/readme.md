# 说明
本镜像为 GuGoTik 提供基础镜像层服务
# 功能  
- 提供 FFmpeg 环境
- 将 static 目录下的 font.ttf 作为 GuGoTik 的水印标记
# 集成
本镜像推送后，将作为主 Dockerfile 的 prod 基础镜像，如果修改了请同步修改主 Dockerfile