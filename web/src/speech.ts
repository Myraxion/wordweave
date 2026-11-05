import { ref } from 'vue'

export const speechError = ref('')
let generation = 0

export function stopSpeech() {
  generation++
  if ('speechSynthesis' in window) window.speechSynthesis.cancel()
}

export function playSpeech(text: string) {
  stopSpeech()
  speechError.value = ''
  if (!('speechSynthesis' in window) || !('SpeechSynthesisUtterance' in window)) {
    speechError.value = '当前设备暂无可用的美式英语声音'
    return
  }
  const synth = window.speechSynthesis
  const current = generation
  const play = () => {
    if (current !== generation) return
    const voice = synth.getVoices().find(item => item.lang.toLowerCase() === 'en-us')
    if (!voice) {
      speechError.value = '当前设备暂无可用的美式英语声音'
      return
    }
    const utterance = new SpeechSynthesisUtterance(text)
    utterance.lang = 'en-US'
    utterance.voice = voice
    utterance.onerror = () => { if (current === generation) speechError.value = '发音播放失败，请重试' }
    synth.speak(utterance)
  }
  if (synth.getVoices().length) play()
  else {
    const onVoices = () => {
      if (!synth.getVoices().length) return
      synth.removeEventListener('voiceschanged', onVoices)
      clearTimeout(timer)
      play()
    }
    synth.addEventListener('voiceschanged', onVoices)
    const timer = window.setTimeout(() => {
      synth.removeEventListener('voiceschanged', onVoices)
      if (current === generation) speechError.value = '声音暂不可用，请重试'
    }, 1500)
  }
}
