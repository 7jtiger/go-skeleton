#!/bin/bash

# MS-Gateway P2P 화상채팅 테스트 미디어 파일 생성 스크립트
# FFmpeg를 사용하여 VP8 비디오(IVF)와 Opus 오디오(OGG) 파일을 생성합니다.

echo "========================================="
echo "🎬 테스트 미디어 파일 생성"
echo "========================================="
echo ""

# FFmpeg 설치 확인
if ! command -v ffmpeg &> /dev/null; then
    echo "❌ FFmpeg가 설치되어 있지 않습니다."
    echo ""
    echo "설치 방법:"
    echo "  Ubuntu/Debian: sudo apt-get install ffmpeg"
    echo "  macOS:         brew install ffmpeg"
    echo "  Windows:       https://ffmpeg.org/download.html"
    exit 1
fi

echo "✅ FFmpeg 설치 확인됨"
echo ""

# 출력 디렉토리 생성
MEDIA_DIR="test_media"
mkdir -p "$MEDIA_DIR"

echo "📁 미디어 파일 저장 디렉토리: $MEDIA_DIR"
echo ""

# 1. 테스트 패턴 비디오 생성 (640x480, 30fps, 10초)
echo "1️⃣  테스트 패턴 비디오 생성 중..."
ffmpeg -f lavfi -i testsrc=duration=10:size=640x480:rate=30 \
    -c:v libvpx -b:v 1M \
    "$MEDIA_DIR/test_video.ivf" -y \
    -loglevel warning

if [ $? -eq 0 ]; then
    echo "   ✅ test_video.ivf 생성 완료 (640x480@30fps, 10초)"
else
    echo "   ❌ 비디오 생성 실패"
fi
echo ""

# 2. 컬러 바 비디오 생성 (고품질)
echo "2️⃣  컬러 바 비디오 생성 중..."
ffmpeg -f lavfi -i smptebars=duration=10:size=640x480:rate=30 \
    -c:v libvpx -b:v 1M \
    "$MEDIA_DIR/colorbar_video.ivf" -y \
    -loglevel warning

if [ $? -eq 0 ]; then
    echo "   ✅ colorbar_video.ivf 생성 완료"
else
    echo "   ❌ 컬러 바 비디오 생성 실패"
fi
echo ""

# 3. 사인파 오디오 생성 (440Hz, 10초)
echo "3️⃣  사인파 오디오 생성 중..."
ffmpeg -f lavfi -i sine=frequency=440:duration=10 \
    -c:a libopus -b:a 48k \
    "$MEDIA_DIR/test_audio.ogg" -y \
    -loglevel warning

if [ $? -eq 0 ]; then
    echo "   ✅ test_audio.ogg 생성 완료 (440Hz, 10초)"
else
    echo "   ❌ 오디오 생성 실패"
fi
echo ""

# 4. 음성 합성 오디오 생성 (다양한 주파수)
echo "4️⃣  음성 합성 오디오 생성 중..."
ffmpeg -f lavfi -i "sine=frequency=523:duration=2,sine=frequency=587:duration=2,sine=frequency=659:duration=2,sine=frequency=698:duration=2,sine=frequency=784:duration=2" \
    -c:a libopus -b:a 48k \
    "$MEDIA_DIR/melody_audio.ogg" -y \
    -loglevel warning

if [ $? -eq 0 ]; then
    echo "   ✅ melody_audio.ogg 생성 완료 (멜로디 패턴)"
else
    echo "   ❌ 멜로디 오디오 생성 실패"
fi
echo ""

# 5. 웹캠/마이크에서 캡처 (선택적)
echo "5️⃣  웹캠/마이크에서 캡처하시겠습니까? (y/N)"
read -t 5 -n 1 CAPTURE_CHOICE
echo ""

if [ "$CAPTURE_CHOICE" = "y" ] || [ "$CAPTURE_CHOICE" = "Y" ]; then
    echo "   📹 웹캠에서 비디오 캡처 중 (5초)..."
    
    # Linux (V4L2)
    if [ -e "/dev/video0" ]; then
        ffmpeg -f v4l2 -i /dev/video0 \
            -c:v libvpx -b:v 1M -r 30 -t 5 \
            "$MEDIA_DIR/webcam_video.ivf" -y \
            -loglevel warning 2>/dev/null
        
        if [ $? -eq 0 ]; then
            echo "   ✅ webcam_video.ivf 생성 완료"
        else
            echo "   ⚠️  웹캠 캡처 실패 (권한 또는 장치 문제)"
        fi
    else
        echo "   ⚠️  /dev/video0 장치를 찾을 수 없습니다"
    fi
    
    echo ""
    echo "   🎤 마이크에서 오디오 캡처 중 (5초)..."
    
    # Linux (ALSA)
    if command -v arecord &> /dev/null; then
        ffmpeg -f alsa -i default \
            -c:a libopus -b:a 48k -t 5 \
            "$MEDIA_DIR/mic_audio.ogg" -y \
            -loglevel warning 2>/dev/null
        
        if [ $? -eq 0 ]; then
            echo "   ✅ mic_audio.ogg 생성 완료"
        else
            echo "   ⚠️  마이크 캡처 실패"
        fi
    else
        echo "   ⚠️  오디오 캡처 장치를 찾을 수 없습니다"
    fi
else
    echo "   ⏭️  웹캠/마이크 캡처 건너뜀"
fi

echo ""
echo "========================================="
echo "✅ 미디어 파일 생성 완료!"
echo "========================================="
echo ""
echo "📂 생성된 파일:"
ls -lh "$MEDIA_DIR"/*.{ivf,ogg} 2>/dev/null | awk '{printf "   %s (%s)\n", $9, $5}'
echo ""
echo "🚀 사용 예시:"
echo "   ./video-chat-av alice ws://localhost:8080/webrtc/v01/ws \\"
echo "                   $MEDIA_DIR/test_video.ivf $MEDIA_DIR/test_audio.ogg"
echo ""
echo "   ./video-chat-av bob ws://localhost:8080/webrtc/v01/ws \\"
echo "                   $MEDIA_DIR/colorbar_video.ivf $MEDIA_DIR/melody_audio.ogg"
echo ""

