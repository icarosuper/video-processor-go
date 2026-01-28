# Test Data

Este diretório contém arquivos de teste para os testes unitários.

## Gerando vídeo de teste

Para gerar um vídeo de teste válido, use FFmpeg:

```bash
# Gerar vídeo de teste (5 segundos, 640x480, com áudio)
ffmpeg -f lavfi -i testsrc=duration=5:size=640x480:rate=30 \
       -f lavfi -i sine=frequency=1000:duration=5 \
       -pix_fmt yuv420p \
       test_video.mp4
```

Os testes criarão vídeos de teste automaticamente quando necessário.
