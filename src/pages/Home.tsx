import React, { useState, useRef, useEffect } from 'react';
import { Card, Button, Select, Typography, Space, Progress, message } from 'antd';
import { AudioOutlined, PlayCircleOutlined, PauseCircleOutlined, StopOutlined, CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { createAudioProcessor, audioDataToWavBase64 } from '../utils/audioUtils';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;

interface RecognitionResult {
  type: string;
  text: string;
  is_final: boolean;
  confidence: number;
  timestamp: number;
}

interface AudioDevice {
  deviceId: string;
  label: string;
}

export default function Home() {
  const [isRecording, setIsRecording] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [recognitionText, setRecognitionText] = useState('');
  const [fullText, setFullText] = useState('');
  const [audioDevices, setAudioDevices] = useState<AudioDevice[]>([]);
  const [selectedDevice, setSelectedDevice] = useState<string>('');
  const [recordingDuration, setRecordingDuration] = useState(0);
  const [wsConnected, setWsConnected] = useState(false);
  
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const audioChunksRef = useRef<Blob[]>([]);
  const recordedAudioRef = useRef<Blob | null>(null);
  const audioElementRef = useRef<HTMLAudioElement | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const durationIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const sequenceRef = useRef(1);
  const lastChunkRef = useRef<string | null>(null);
  const audioProcessorRef = useRef<any>(null);
  const audioDataBufferRef = useRef<Float32Array[]>([]);

  // 获取音频设备列表
  useEffect(() => {
    const getAudioDevices = async () => {
      try {
        const devices = await navigator.mediaDevices.enumerateDevices();
        const audioInputs = devices
          .filter(device => device.kind === 'audioinput')
          .map(device => ({
            deviceId: device.deviceId,
            label: device.label || `麦克风 ${device.deviceId.slice(0, 8)}`
          }));
        setAudioDevices(audioInputs);
        if (audioInputs.length > 0) {
          setSelectedDevice(audioInputs[0].deviceId);
        }
      } catch (error) {
        console.error('获取音频设备失败:', error);
        message.error('无法获取音频设备');
      }
    };

    getAudioDevices();
  }, []);

  // WebSocket连接
  const connectWebSocket = () => {
    const ws = new WebSocket('ws://localhost:8080/ws');
    
    ws.onopen = () => {
      console.log('WebSocket连接已建立');
      setWsConnected(true);
      
      // 发送开始识别消息
      const startMessage = {
        type: 'start_recognition',
        audio_config: {
          sample_rate: 16000,
          channels: 1,
          format: 'wav'
        }
      };
      ws.send(JSON.stringify(startMessage));
    };
    
    ws.onmessage = (event) => {
      try {
        const result: RecognitionResult = JSON.parse(event.data);
        if (result.type === 'recognition_result') {
          if (result.is_final) {
            setFullText(prev => prev + result.text + ' ');
            setRecognitionText('');
          } else {
            setRecognitionText(result.text);
          }
        }
      } catch (error) {
        console.error('解析识别结果失败:', error);
      }
    };
    
    ws.onclose = () => {
      console.log('WebSocket连接已关闭');
      setWsConnected(false);
    };
    
    ws.onerror = (error) => {
      console.error('WebSocket错误:', error);
      message.error('WebSocket连接失败');
      setWsConnected(false);
    };
    
    wsRef.current = ws;
  };

  // 开始录音
  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          deviceId: selectedDevice ? { exact: selectedDevice } : undefined,
          sampleRate: 16000,
          channelCount: 1,
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true
        }
      });

      // 连接WebSocket
      connectWebSocket();

      // 初始化音频数据缓冲区
      audioChunksRef.current = [];
      audioDataBufferRef.current = [];
      sequenceRef.current = 1;

      // 创建音频处理器用于实时处理
      const processor = createAudioProcessor(stream, (audioData) => {
        // 存储音频数据用于后续播放
        audioDataBufferRef.current.push(new Float32Array(audioData));
        
        // 转换为PCM Base64（不包含WAV头）并发送到WebSocket
        const pcmBase64 = audioDataToWavBase64(audioData, 16000, 1, false);
        lastChunkRef.current = pcmBase64;
        
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          const audioMessage = {
            type: 'audio_data',
            data: pcmBase64,
            sequence: sequenceRef.current++,
            is_final: false
          };
          wsRef.current.send(JSON.stringify(audioMessage));
        }
      });
      
      audioProcessorRef.current = processor;

      // 同时使用MediaRecorder录制完整音频用于播放
      const mediaRecorder = new MediaRecorder(stream, {
        mimeType: 'audio/webm;codecs=opus'
      });

      mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          audioChunksRef.current.push(event.data);
        }
      };

      mediaRecorder.onstop = () => {
        const audioBlob = new Blob(audioChunksRef.current, { type: 'audio/webm' });
        recordedAudioRef.current = audioBlob;
        
        // 在停止前发送最后一包，标记is_final=true
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          const finalData = lastChunkRef.current || '';
          const finalMessage = {
            type: 'audio_data',
            data: finalData,
            sequence: sequenceRef.current++,
            is_final: true
          };
          wsRef.current.send(JSON.stringify(finalMessage));
        }
        
        // 发送结束消息
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          const stopMessage = {
            type: 'stop_recognition'
          };
          wsRef.current.send(JSON.stringify(stopMessage));
          wsRef.current.close();
        }
        
        // 停止音频处理器
        if (audioProcessorRef.current) {
          audioProcessorRef.current.stop();
          audioProcessorRef.current = null;
        }
        
        stream.getTracks().forEach(track => track.stop());
      };

      mediaRecorderRef.current = mediaRecorder;
      mediaRecorder.start(200); // 每200ms发送一次数据
      setIsRecording(true);
      setRecordingDuration(0);
      
      // 开始计时
      durationIntervalRef.current = setInterval(() => {
        setRecordingDuration(prev => prev + 1);
      }, 1000);
      
    } catch (error) {
      console.error('开始录音失败:', error);
      message.error('无法访问麦克风，请检查麦克风权限');
    }
  };

  // 停止录音
  const stopRecording = () => {
    if (mediaRecorderRef.current && isRecording) {
      mediaRecorderRef.current.stop();
      setIsRecording(false);
      
      if (durationIntervalRef.current) {
        clearInterval(durationIntervalRef.current);
        durationIntervalRef.current = null;
      }
    }
  };

  // 播放录音
  const playRecording = () => {
    if (recordedAudioRef.current) {
      const audioUrl = URL.createObjectURL(recordedAudioRef.current);
      const audio = new Audio(audioUrl);
      
      audio.onplay = () => setIsPlaying(true);
      audio.onended = () => setIsPlaying(false);
      audio.onpause = () => setIsPlaying(false);
      
      audioElementRef.current = audio;
      audio.play();
    }
  };

  // 暂停播放
  const pausePlayback = () => {
    if (audioElementRef.current) {
      audioElementRef.current.pause();
      setIsPlaying(false);
    }
  };

  // 复制文本
  const copyText = () => {
    const textToCopy = fullText + recognitionText;
    navigator.clipboard.writeText(textToCopy).then(() => {
      message.success('文本已复制到剪贴板');
    }).catch(() => {
      message.error('复制失败');
    });
  };

  // 导出文本
  const exportText = () => {
    const textToExport = fullText + recognitionText;
    const blob = new Blob([textToExport], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `语音识别结果_${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    message.success('文本已导出');
  };

  // 清空历史
  const clearHistory = () => {
    setFullText('');
    setRecognitionText('');
    message.success('历史记录已清空');
  };

  // 格式化时间
  const formatDuration = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  };

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-4xl mx-auto">
        <Title level={2} className="text-center mb-8">
          🎤 实时语音识别
        </Title>
        
        {/* 麦克风选择器 */}
        <Card className="mb-6">
          <Space direction="vertical" className="w-full">
            <Text strong>选择麦克风设备：</Text>
            <Select
              value={selectedDevice}
              onChange={setSelectedDevice}
              className="w-full"
              placeholder="请选择麦克风"
              disabled={isRecording}
            >
              {audioDevices.map(device => (
                <Option key={device.deviceId} value={device.deviceId}>
                  <AudioOutlined className="mr-2" />
                  {device.label}
                </Option>
              ))}
            </Select>
          </Space>
        </Card>

        {/* 录音控制 */}
        <Card className="mb-6 text-center">
          <Space direction="vertical" size="large" className="w-full">
            <div>
              <Button
                type={isRecording ? "danger" : "primary"}
                size="large"
                shape="circle"
                icon={isRecording ? <StopOutlined /> : <AudioOutlined />}
                onClick={isRecording ? stopRecording : startRecording}
                className="w-20 h-20 text-2xl"
                disabled={!selectedDevice}
              />
            </div>
            
            <div>
              <Text strong className="text-lg">
                {isRecording ? '录音中...' : '点击开始录音'}
              </Text>
              {isRecording && (
                <div className="mt-2">
                  <Text type="secondary">
                    录音时长: {formatDuration(recordingDuration)}
                  </Text>
                  <br />
                  <Text type={wsConnected ? "success" : "danger"}>
                    WebSocket: {wsConnected ? '已连接' : '未连接'}
                  </Text>
                </div>
              )}
            </div>
          </Space>
        </Card>

        {/* 实时文字显示 */}
        <Card title="识别结果" className="mb-6">
          <div className="min-h-32 p-4 bg-white border rounded-lg">
            <Paragraph className="text-lg leading-relaxed">
              {fullText}
              <Text mark className="bg-yellow-200">
                {recognitionText}
              </Text>
              {(fullText || recognitionText) && (
                <span className="inline-block w-2 h-6 bg-blue-500 ml-1 animate-pulse" />
              )}
            </Paragraph>
            
            {!fullText && !recognitionText && (
              <Text type="secondary" className="italic">
                开始录音后，识别的文字将在这里实时显示...
              </Text>
            )}
          </div>
        </Card>

        {/* 音频播放和操作 */}
        <Card title="录音回放与操作" className="mb-6">
          <Space wrap>
            <Button
              icon={isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              onClick={isPlaying ? pausePlayback : playRecording}
              disabled={!recordedAudioRef.current}
            >
              {isPlaying ? '暂停播放' : '播放录音'}
            </Button>
            
            <Button
              icon={<CopyOutlined />}
              onClick={copyText}
              disabled={!fullText && !recognitionText}
            >
              复制文本
            </Button>
            
            <Button
              icon={<DownloadOutlined />}
              onClick={exportText}
              disabled={!fullText && !recognitionText}
            >
              导出文本
            </Button>
            
            <Button
              danger
              onClick={clearHistory}
              disabled={!fullText && !recognitionText}
            >
              清空历史
            </Button>
          </Space>
        </Card>

        {/* 使用说明 */}
        <Card title="使用说明" size="small">
          <ul className="text-sm text-gray-600 space-y-1">
            <li>• 选择合适的麦克风设备</li>
            <li>• 点击录音按钮开始语音识别</li>
            <li>• 说话时文字会实时显示，黄色高亮为临时识别结果</li>
            <li>• 停止录音后可以重播音频</li>
            <li>• 支持复制和导出识别的文本</li>
          </ul>
        </Card>
      </div>
    </div>
  );
}