/**
 * 音频处理工具函数
 */

/**
 * 将AudioBuffer转换为WAV格式的ArrayBuffer
 * @param {AudioBuffer} audioBuffer - 音频缓冲区
 * @param {number} sampleRate - 采样率
 * @returns {ArrayBuffer} WAV格式的ArrayBuffer
 */
export function audioBufferToWav(audioBuffer, sampleRate = 16000) {
  const numberOfChannels = audioBuffer.numberOfChannels;
  const length = audioBuffer.length * numberOfChannels * 2; // 16位音频
  const buffer = new ArrayBuffer(44 + length);
  const view = new DataView(buffer);
  
  // WAV文件头
  const writeString = (offset, string) => {
    for (let i = 0; i < string.length; i++) {
      view.setUint8(offset + i, string.charCodeAt(i));
    }
  };
  
  // RIFF标识符
  writeString(0, 'RIFF');
  // 文件长度
  view.setUint32(4, 36 + length, true);
  // WAVE标识符
  writeString(8, 'WAVE');
  // fmt子块
  writeString(12, 'fmt ');
  // fmt子块长度
  view.setUint32(16, 16, true);
  // 音频格式 (PCM)
  view.setUint16(20, 1, true);
  // 声道数
  view.setUint16(22, numberOfChannels, true);
  // 采样率
  view.setUint32(24, sampleRate, true);
  // 字节率
  view.setUint32(28, sampleRate * numberOfChannels * 2, true);
  // 块对齐
  view.setUint16(32, numberOfChannels * 2, true);
  // 位深
  view.setUint16(34, 16, true);
  // data子块
  writeString(36, 'data');
  // data子块长度
  view.setUint32(40, length, true);
  
  // 写入音频数据
  let offset = 44;
  for (let i = 0; i < audioBuffer.length; i++) {
    for (let channel = 0; channel < numberOfChannels; channel++) {
      const sample = Math.max(-1, Math.min(1, audioBuffer.getChannelData(channel)[i]));
      view.setInt16(offset, sample < 0 ? sample * 0x8000 : sample * 0x7FFF, true);
      offset += 2;
    }
  }
  
  return buffer;
}

/**
 * 将Blob转换为AudioBuffer
 * @param {Blob} blob - 音频Blob
 * @param {AudioContext} audioContext - 音频上下文
 * @returns {Promise<AudioBuffer>}
 */
export async function blobToAudioBuffer(blob, audioContext) {
  const arrayBuffer = await blob.arrayBuffer();
  return await audioContext.decodeAudioData(arrayBuffer);
}

/**
 * 重采样AudioBuffer到指定采样率
 * @param {AudioBuffer} audioBuffer - 原始音频缓冲区
 * @param {number} targetSampleRate - 目标采样率
 * @returns {AudioBuffer} 重采样后的AudioBuffer
 */
export function resampleAudioBuffer(audioBuffer, targetSampleRate) {
  const audioContext = new (window.AudioContext || window.webkitAudioContext)();
  const ratio = audioBuffer.sampleRate / targetSampleRate;
  const newLength = Math.round(audioBuffer.length / ratio);
  const newBuffer = audioContext.createBuffer(audioBuffer.numberOfChannels, newLength, targetSampleRate);
  
  for (let channel = 0; channel < audioBuffer.numberOfChannels; channel++) {
    const oldData = audioBuffer.getChannelData(channel);
    const newData = newBuffer.getChannelData(channel);
    
    for (let i = 0; i < newLength; i++) {
      const index = i * ratio;
      const indexFloor = Math.floor(index);
      const indexCeil = Math.min(indexFloor + 1, oldData.length - 1);
      const fraction = index - indexFloor;
      
      // 线性插值
      newData[i] = oldData[indexFloor] * (1 - fraction) + oldData[indexCeil] * fraction;
    }
  }
  
  return newBuffer;
}

/**
 * 将音频数据转换为Base64编码的WAV格式
 * @param {Float32Array} audioData - 音频数据
 * @param {number} sampleRate - 采样率
 * @param {number} channels - 声道数
 * @param {boolean} includeWavHeader - 是否包含WAV头，默认true
 * @returns {string} Base64编码的WAV数据或PCM数据
 */
export function audioDataToWavBase64(audioData, sampleRate = 16000, channels = 1, includeWavHeader = true) {
  const length = audioData.length * channels * 2;
  const bufferSize = includeWavHeader ? 44 + length : length;
  const buffer = new ArrayBuffer(bufferSize);
  const view = new DataView(buffer);
  
  let offset = 0;
  
  if (includeWavHeader) {
    // WAV文件头
    const writeString = (offset, string) => {
      for (let i = 0; i < string.length; i++) {
        view.setUint8(offset + i, string.charCodeAt(i));
      }
    };
    
    writeString(0, 'RIFF');
    view.setUint32(4, 36 + length, true);
    writeString(8, 'WAVE');
    writeString(12, 'fmt ');
    view.setUint32(16, 16, true);
    view.setUint16(20, 1, true);
    view.setUint16(22, channels, true);
    view.setUint32(24, sampleRate, true);
    view.setUint32(28, sampleRate * channels * 2, true);
    view.setUint16(32, channels * 2, true);
    view.setUint16(34, 16, true);
    writeString(36, 'data');
    view.setUint32(40, length, true);
    
    offset = 44;
  }
  
  // 写入音频数据
  for (let i = 0; i < audioData.length; i++) {
    const sample = Math.max(-1, Math.min(1, audioData[i]));
    view.setInt16(offset, sample < 0 ? sample * 0x8000 : sample * 0x7FFF, true);
    offset += 2;
  }
  
  // 转换为Base64
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  
  return btoa(binary);
}

/**
 * 创建音频处理器，用于实时处理音频流
 * @param {MediaStream} stream - 媒体流
 * @param {Function} onAudioData - 音频数据回调
 * @returns {Object} 音频处理器对象
 */
export function createAudioProcessor(stream, onAudioData) {
  const audioContext = new (window.AudioContext || window.webkitAudioContext)({
    sampleRate: 16000
  });
  
  const source = audioContext.createMediaStreamSource(stream);
  const processor = audioContext.createScriptProcessor(4096, 1, 1);
  
  processor.onaudioprocess = (event) => {
    const inputBuffer = event.inputBuffer;
    const audioData = inputBuffer.getChannelData(0);
    onAudioData(new Float32Array(audioData));
  };
  
  source.connect(processor);
  processor.connect(audioContext.destination);
  
  return {
    audioContext,
    source,
    processor,
    stop: () => {
      processor.disconnect();
      source.disconnect();
      audioContext.close();
    }
  };
}