import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { dicts, LANGS, type Key, type Lang } from './i18n'

// ---------- API ----------
export class ApiError extends Error {
  status: number
  constructor(status: number, msg: string) { super(msg); this.status = status }
}

export async function api<T = any>(path: string, opts: { method?: string; body?: unknown; form?: FormData } = {}): Promise<T> {
  const res = await fetch('/api' + path, {
    method: opts.method ?? (opts.body || opts.form ? 'POST' : 'GET'),
    headers: opts.body ? { 'Content-Type': 'application/json' } : undefined,
    body: opts.form ?? (opts.body ? JSON.stringify(opts.body) : undefined),
    credentials: 'same-origin',
  })
  if (!res.ok) {
    let msg = ''
    try { msg = (await res.json()).error } catch { /* not json */ }
    throw new ApiError(res.status, msg || `Request failed (${res.status})`)
  }
  const ct = res.headers.get('content-type') || ''
  return (ct.includes('json') ? res.json() : res.blob()) as Promise<T>
}

// ---------- routing ----------
export function useRoute() {
  const [path, setPath] = useState(location.pathname)
  useEffect(() => {
    const on = () => setPath(location.pathname)
    addEventListener('popstate', on)
    return () => removeEventListener('popstate', on)
  }, [])
  return path
}
export function navigate(to: string) {
  if (to === location.pathname) return
  history.pushState(null, '', to)
  dispatchEvent(new PopStateEvent('popstate'))
  scrollTo({ top: 0 })
}
export function Link({ to, children, ...rest }: { to: string; children: ReactNode } & React.AnchorHTMLAttributes<HTMLAnchorElement>) {
  return (
    <a href={to} {...rest} onClick={(e) => { if (e.metaKey || e.ctrlKey) return; e.preventDefault(); rest.onClick?.(e); navigate(to) }}>
      {children}
    </a>
  )
}

// ---------- app context ----------
export type User = { id: number; phone: string; name: string; lang: Lang; role: string; prefs: Record<string, any> }

type Ctx = {
  user: User | null
  setUser: (u: User | null) => void
  lang: Lang
  setLang: (l: Lang) => void
  t: (k: Key, vars?: Record<string, string | number>) => string
  theme: 'light' | 'dark'
  toggleTheme: () => void
  toast: (msg: string) => void
}
const AppCtx = createContext<Ctx>(null!)
export const useApp = () => useContext(AppCtx)

export function AppProvider({ children }: { children: ReactNode }) {
  const [user, setUserState] = useState<User | null>(null)
  const [lang, setLangState] = useState<Lang>(() => {
    try { return (localStorage.getItem('finro-lang') as Lang) || (localStorage.getItem('munshi-lang') as Lang) || 'en' } catch { return 'en' }
  })
  const [theme, setTheme] = useState<'light' | 'dark'>(() => (document.documentElement.dataset.theme as 'light' | 'dark') || 'light')
  const [toastMsg, setToastMsg] = useState('')

  useEffect(() => {
    document.documentElement.lang = lang
    try { localStorage.setItem('finro-lang', lang) } catch { /* private mode */ }
  }, [lang])
  useEffect(() => {
    document.documentElement.dataset.theme = theme
    document.querySelector('meta[name="theme-color"]')?.setAttribute('content', theme === 'dark' ? '#121827' : '#F4EDE0')
    try { localStorage.setItem('finro-theme', theme) } catch { /* private mode */ }
  }, [theme])

  const setUser = useCallback((u: User | null) => {
    setUserState(u)
    if (u) setLangState(u.lang)
  }, [])
  const setLang = useCallback((l: Lang) => {
    setLangState(l)
    setUserState((u) => {
      if (u && u.lang !== l) api('/settings', { method: 'PUT', body: { lang: l } }).catch(() => {})
      return u ? { ...u, lang: l } : u
    })
  }, [])
  const t = useCallback((k: Key, vars?: Record<string, string | number>) => {
    let s: string = dicts[lang][k] ?? dicts.en[k] ?? k
    if (vars) for (const [vk, vv] of Object.entries(vars)) s = s.replace(`{${vk}}`, String(vv))
    return s
  }, [lang])
  const toast = useCallback((msg: string) => {
    setToastMsg(msg)
    setTimeout(() => setToastMsg((m) => (m === msg ? '' : m)), 3200)
  }, [])

  return (
    <AppCtx.Provider value={{ user, setUser, lang, setLang, t, theme, toggleTheme: () => setTheme((x) => (x === 'dark' ? 'light' : 'dark')), toast }}>
      {children}
      {toastMsg && <div className="toast" role="status">{toastMsg}</div>}
    </AppCtx.Provider>
  )
}

// ---------- data hook ----------
export function useData<T>(path: string | null, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(!!path)
  const load = useCallback(async () => {
    if (!path) return
    setLoading(true)
    setError('')
    try { setData(await api<T>(path)) } catch (e) { setError((e as Error).message) } finally { setLoading(false) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path, ...deps])
  useEffect(() => { load() }, [load])
  return { data, setData, error, loading, reload: load }
}

// ---------- formatting ----------
export const rupees = (paise: number) => '₹' + Math.round(paise / 100).toLocaleString('en-IN')
export const rupeesInr = (inr: number) => '₹' + Math.round(inr).toLocaleString('en-IN')

export function ago(iso: string | null | undefined, lang: Lang) {
  if (!iso) return '—'
  const s = (Date.now() - new Date(iso).getTime()) / 1000
  const rtf = new Intl.RelativeTimeFormat(LANGS.find((l) => l.code === lang)?.speech ?? 'en-IN', { numeric: 'auto' })
  if (s < 60) return rtf.format(-Math.round(s), 'second')
  if (s < 3600) return rtf.format(-Math.round(s / 60), 'minute')
  if (s < 86400) return rtf.format(-Math.round(s / 3600), 'hour')
  return rtf.format(-Math.round(s / 86400), 'day')
}

export function fmtDate(d: string | null | undefined, lang: Lang) {
  if (!d) return ''
  return new Date(d + 'T00:00:00').toLocaleDateString(LANGS.find((l) => l.code === lang)?.speech, { day: 'numeric', month: 'short', year: 'numeric' })
}

// ---------- tiny markdown (bold, italics, lists, paragraphs) ----------
function inline(s: string, key: string): ReactNode[] {
  const parts = s.split(/(\*\*[^*]+\*\*|\*[^*]+\*|\[NEEDED:[^\]]*\])/g)
  return parts.map((p, i) => {
    if (p.startsWith('**') && p.endsWith('**')) return <strong key={key + i}>{p.slice(2, -2)}</strong>
    if (p.startsWith('[NEEDED')) return <span key={key + i} className="needed">{p}</span>
    if (p.startsWith('*') && p.endsWith('*') && p.length > 2) return <em key={key + i}>{p.slice(1, -1)}</em>
    return p
  })
}
export function Md({ text }: { text: string }) {
  const blocks: ReactNode[] = []
  const lines = text.split('\n')
  let list: { ordered: boolean; items: string[] } | null = null
  const flush = () => {
    if (!list) return
    const Tag = list.ordered ? 'ol' : 'ul'
    blocks.push(<Tag key={blocks.length}>{list.items.map((it, i) => <li key={i}>{inline(it, `l${blocks.length}-${i}`)}</li>)}</Tag>)
    list = null
  }
  for (const raw of lines) {
    const line = raw.trim()
    const ol = line.match(/^\d+[.)]\s+(.*)/)
    const ul = line.match(/^[-•*]\s+(.*)/)
    if (ol || ul) {
      const ordered = !!ol
      if (!list || list.ordered !== ordered) { flush(); list = { ordered, items: [] } }
      list.items.push((ol ?? ul)![1])
    } else {
      flush()
      if (line) blocks.push(<p key={blocks.length}>{inline(line.replace(/^#+\s*/, ''), `p${blocks.length}`)}</p>)
    }
  }
  flush()
  return <>{blocks}</>
}

// ---------- voice: record → server transcription, fallback to Web Speech ----------
export function useVoice(onText: (text: string) => void) {
  const { lang, toast, t } = useApp()
  const [recording, setRecording] = useState(false)
  const [busy, setBusy] = useState(false)
  const rec = useRef<MediaRecorder | null>(null)
  const recog = useRef<any>(null)

  const webSpeech = () => {
    const SR = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition
    if (!SR) { toast(t('err_generic')); return }
    const r = new SR()
    r.lang = LANGS.find((l) => l.code === lang)?.speech ?? 'en-IN'
    r.interimResults = false
    r.onresult = (e: any) => onText(e.results[0][0].transcript)
    r.onend = () => setRecording(false)
    r.onerror = () => setRecording(false)
    recog.current = r
    r.start()
    setRecording(true)
  }

  const start = async () => {
    // Browser speech recognition first (instant, supports hi/te/ta in Chrome); server transcription otherwise.
    if (hasWebSpeech() || !navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === 'undefined') return webSpeech()
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mr = new MediaRecorder(stream)
      const chunks: Blob[] = []
      mr.ondataavailable = (e) => e.data.size && chunks.push(e.data)
      mr.onstop = async () => {
        stream.getTracks().forEach((tr) => tr.stop())
        setRecording(false)
        const blob = new Blob(chunks, { type: mr.mimeType || 'audio/webm' })
        if (blob.size < 1200) return
        setBusy(true)
        const fd = new FormData()
        fd.append('audio', blob, 'speech.' + (mr.mimeType.includes('mp4') ? 'mp4' : 'webm'))
        try {
          const { text } = await api<{ text: string }>('/voice/transcribe', { form: fd })
          if (text.trim()) onText(text.trim())
        } catch {
          toast(t('err_generic'))
        } finally { setBusy(false) }
      }
      rec.current = mr
      mr.start()
      setRecording(true)
      setTimeout(() => mr.state === 'recording' && mr.stop(), 90_000)
    } catch {
      webSpeech()
    }
  }
  const stop = () => {
    if (rec.current?.state === 'recording') rec.current.stop()
    recog.current?.stop()
  }
  return { recording, busy, toggle: () => (recording ? stop() : start()) }
}

// ---------- speech output: server TTS, fallback to speechSynthesis ----------
let currentAudio: HTMLAudioElement | null = null
export async function speakText(text: string, lang: Lang, onEnd?: () => void) {
  stopSpeaking()
  const clean = text.replace(/\*\*|\*|#/g, '')
  try {
    if (sessionFlag('finro-no-server-tts')) throw new Error('server tts unavailable')
    const blob = await api<Blob>('/voice/tts', { body: { text: clean } })
    const a = new Audio(URL.createObjectURL(blob))
    currentAudio = a
    a.onended = () => onEnd?.()
    await a.play()
  } catch (e) {
    if (e instanceof ApiError) sessionFlag('finro-no-server-tts', true)
    const u = new SpeechSynthesisUtterance(clean)
    u.lang = LANGS.find((l) => l.code === lang)?.speech ?? 'en-IN'
    u.onend = () => onEnd?.()
    speechSynthesis.speak(u)
  }
}
export function stopSpeaking() {
  currentAudio?.pause()
  currentAudio = null
  if ('speechSynthesis' in window) speechSynthesis.cancel()
}

export function fileToDataURL(file: File, maxSide = 1400): Promise<string> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => {
      const scale = Math.min(1, maxSide / Math.max(img.width, img.height))
      const c = document.createElement('canvas')
      c.width = Math.round(img.width * scale)
      c.height = Math.round(img.height * scale)
      c.getContext('2d')!.drawImage(img, 0, 0, c.width, c.height)
      URL.revokeObjectURL(img.src)
      resolve(c.toDataURL('image/jpeg', 0.82))
    }
    img.onerror = reject
    img.src = URL.createObjectURL(file)
  })
}

function hasWebSpeech() {
  return !!((window as any).SpeechRecognition || (window as any).webkitSpeechRecognition)
}

// sessionFlag reads (or sets) a per-tab flag; storage failures just mean "not set".
function sessionFlag(key: string, set?: boolean) {
  try {
    if (set) sessionStorage.setItem(key, '1')
    return sessionStorage.getItem(key) === '1'
  } catch { return false }
}

export const LANGS_SPEECH = (lang: string) => LANGS.find((l) => l.code === lang)?.speech ?? 'en-IN'
