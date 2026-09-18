import { useEffect, useRef, useState } from 'react'
import type { Key } from './i18n'
import { api, fileToDataURL, Link, Md, navigate, speakText, useApp, useData } from './lib'
import {
  BlueprintView, DraftSummary, EntriesProposal, FitCard, GrantCard, Icon, MicButton, PatternCard, QRCard, SpeakButton, Spinner, SummaryStats, VerdictCard,
  type Fit, type Grant,
} from './ui'
import { SchemeIntakeForm, SchemeRecs } from './pages/Funding'
import { AdviceCard } from './pages/Stock'

const STAGES = ['safety', 'onboarding', 'records', 'blueprint', 'funding', 'apply'] as const

type Step = { tool: string; label: string; ok: boolean; ms: number }
type Card = { type: string; data: any }
type Msg = { id: number; role: 'user' | 'assistant'; content: string; meta: { steps?: Step[]; cards?: Card[]; nudge?: { label: string; prompt: string; stage: string } | null; stage?: string; image?: boolean }; created_at: string; fresh?: boolean }
type Session = { id: number; title: string; stage: string; summary: string; updated_at: string }

function Cards({ cards, onSend }: { cards: Card[]; onSend: (m: string) => void }) {
  const { t } = useApp()
  const [drafting, setDrafting] = useState(0)
  const draft = async (id: number) => {
    setDrafting(id)
    try { const d = await api('/drafts', { body: { grant_id: id } }); navigate(`/app/apply/${d.id}`) } catch { setDrafting(0) }
  }
  return (
    <div className="stack" style={{ marginTop: '0.6rem' }}>
      {cards.map((c, i) => {
        switch (c.type) {
          case 'scam_verdict': return <VerdictCard key={i} v={c.data} />
          case 'qr': return <QRCard key={i} q={c.data} />
          case 'records_proposal': return <EntriesProposal key={i} entries={c.data.entries} unclear={c.data.unclear} />
          case 'records_summary': return <SummaryStats key={i} s={c.data} />
          case 'blueprint': return (
            <div key={i} className="stack">
              <BlueprintView b={c.data} compact />
              <Link className="btn sm primary" style={{ alignSelf: 'flex-start' }} to="/app/blueprint"><Icon name="flow" size={16} />{t('open_blueprint')}</Link>
            </div>
          )
          case 'stock_advice': return <AdviceCard key={i} advice={c.data} />
          case 'fits': {
            const fits = (c.data as Fit[]).filter((f) => f.verdict !== 'not_a_fit').slice(0, 3)
            const hidden = (c.data as Fit[]).length - fits.length
            return (
              <div key={i} className="stack">
                {fits.map((f) => <FitCard key={f.grant_id} f={f} onDraft={draft} drafting={drafting === f.grant_id} />)}
                <Link className="small" to="/app/matching">{hidden > 0 ? `+${hidden} ${t('not_a_fit')} · ` : ''}{t('nav_matching')} →</Link>
              </div>
            )
          }
          case 'funding_list': return (
            <div key={i} className="stack">
              {(c.data.grants as Grant[]).slice(0, 3).map((g) => <GrantCard key={g.id} g={g} />)}
              <Link className="small" to="/app/funding">{t('nav_funding')} ({c.data.grants.length}) →</Link>
            </div>
          )
          case 'draft': return <DraftSummary key={i} d={c.data} />
          case 'scheme_intake': return (
            <SchemeIntakeForm key={i} intake={c.data}
              onDone={(p) => onSend(Object.entries(p).map(([k, v]) => `${k.replace(/_/g, ' ')}: ${Array.isArray(v) ? v.join(', ') : v}`).join(' · '))} />
          )
          case 'scheme_recs': return <SchemeRecs key={i} res={c.data} />
          case 'scam_patterns': return (
            <div key={i} className="grid2">{c.data.catalogue.slice(0, 2).map((p: any) => <PatternCard key={p.id} p={p} />)}</div>
          )
          default: return null
        }
      })}
    </div>
  )
}

function Trail({ steps, animate }: { steps: Step[]; animate?: boolean }) {
  if (!steps?.length) return null
  return (
    <ul className="trail" aria-label="What Finro did">
      {steps.map((s, i) => (
        <li key={i} className={animate ? 'anim' : ''} style={animate ? { animationDelay: `${i * 180}ms` } : undefined}>
          <span className={s.ok ? 'tick' : 'cross'}>{s.ok ? '✓' : '✗'}</span>
          <span>{s.label}</span>
          <span className="ms">{(s.ms / 1000).toFixed(1)}s</span>
        </li>
      ))}
    </ul>
  )
}

export default function Assistant({ sessionId }: { sessionId?: number }) {
  const { t, user, lang, toast } = useApp()
  const sessions = useData<Session[]>('/sessions')
  const [session, setSession] = useState<Session | null>(null)
  const [msgs, setMsgs] = useState<Msg[]>([])
  const [text, setText] = useState('')
  const [image, setImage] = useState('')
  const [busy, setBusy] = useState(false)
  const [loading, setLoading] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const taRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (!sessionId) { setSession(null); setMsgs([]); return }
    setLoading(true)
    api<{ session: Session; messages: Msg[] }>(`/sessions/${sessionId}`)
      .then((r) => { setSession(r.session); setMsgs(r.messages) })
      .catch(() => navigate('/app'))
      .finally(() => setLoading(false))
  }, [sessionId])

  useEffect(() => { endRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' }) }, [msgs.length, busy])

  const send = async (message: string, source = 'text') => {
    const m = message.trim()
    if ((!m && !image) || busy) return
    const optimistic: Msg = { id: -Date.now(), role: 'user', content: m, meta: { image: !!image }, created_at: '' }
    setMsgs((x) => [...x, optimistic])
    setText('')
    const img = image
    setImage('')
    setBusy(true)
    try {
      const r = await api<{ session: Session; message: Msg; user_message_id: number }>('/chat', { body: { session_id: session?.id ?? 0, message: m, image: img, source } })
      setMsgs((x) => [...x.map((y) => (y.id === optimistic.id ? { ...y, id: r.user_message_id } : y)), { ...r.message, fresh: true }])
      if (!session) {
        setSession(r.session)
        history.replaceState(null, '', `/app/c/${r.session.id}`)
        sessions.reload()
      } else setSession(r.session)
      if (user?.prefs?.auto_read) speakText(r.message.content, lang)
    } catch (e) {
      setMsgs((x) => x.filter((y) => y.id !== optimistic.id))
      setText(m)
      toast((e as Error).message || t('err_generic'))
    } finally {
      setBusy(false)
      taRef.current?.focus()
    }
  }

  const stageIdx = STAGES.indexOf((session?.stage ?? 'safety') as typeof STAGES[number])
  const lastAssistant = [...msgs].reverse().find((m) => m.role === 'assistant')
  const suggestions: Key[] = ['sug_scam', 'sug_qr', 'sug_sales', 'sug_grow']

  return (
    <div className={`chat ${sessions.data?.length ? 'with-rail' : ''}`}>
      <section style={{ minWidth: 0 }}>
        {sessions.data && sessions.data.length > 0 && (
          <div className="chat-mini row no-print" style={{ marginBottom: 8, gap: 6 }}>
            <select className="select grow" style={{ minHeight: 38 }} aria-label={t('conversations')} value={session?.id ?? ''}
              onChange={(e) => { if (!e.target.value) { setSession(null); setMsgs([]) } navigate(e.target.value ? `/app/c/${e.target.value}` : '/app') }}>
              <option value="">{t('new_chat')}</option>
              {sessions.data.map((s) => <option key={s.id} value={s.id}>{s.title}</option>)}
            </select>
            <button className="btn sm icon" aria-label={t('new_chat')} onClick={() => { setSession(null); setMsgs([]); navigate('/app') }}><Icon name="plus" size={16} /></button>
          </div>
        )}
        <div className="stages" role="list" aria-label="Journey">
          {STAGES.map((s, i) => (
            <button key={s} role="listitem" className={`stage-tab ${i < stageIdx ? 'done' : ''}`} aria-current={i === stageIdx ? 'step' : undefined}
              onClick={() => { setText(t(({ safety: 'sug_scam', onboarding: 'sug_qr', records: 'sug_sales', blueprint: 'sug_grow', funding: 'sug_grow', apply: 'sug_grow' } as const)[s])); taRef.current?.focus() }}
              title={t(`stage_${s}` as Key)}>
              <span className="n">{i + 1}</span>{t(`stage_${s}` as Key)}
            </button>
          ))}
        </div>

        <div className="thread" aria-live="polite">
          {loading && <Spinner />}
          {!loading && msgs.length === 0 && (
            <div className="sheet ruled" style={{ lineHeight: '32px', paddingTop: '0.5rem' }}>
              <h2 className="display" style={{ fontSize: '1.9rem', lineHeight: '48px' }}>{t('hello')}, {user?.name.split(' ')[0]}.</h2>
              <p style={{ fontSize: '1.08rem' }}>{t('chat_intro')}</p>
            </div>
          )}
          {msgs.map((m) => (
            <div key={m.id} className={`msg ${m.role}`}>
              {m.role === 'assistant' && <span className="avatar" aria-hidden="true">म</span>}
              <div style={{ minWidth: 0, flex: m.role === 'assistant' ? 1 : undefined }}>
                {m.role === 'assistant' && <Trail steps={m.meta.steps ?? []} animate={m.fresh} />}
                <div className="bubble">
                  {m.meta.image && <div className="chip" style={{ marginBottom: 6 }}><Icon name="camera" size={14} /> image</div>}
                  <Md text={m.content} />
                  {m.role === 'assistant' && m.content && <div className="row" style={{ justifyContent: 'flex-end', marginTop: 2 }}><SpeakButton text={m.content} /></div>}
                </div>
                {m.role === 'assistant' && m.meta.cards && m.meta.cards.length > 0 && <Cards cards={m.meta.cards} onSend={(s) => send(s, 'chip')} />}
                {m.role === 'assistant' && m.meta.nudge && m === lastAssistant && !busy && (
                  <button className="nudge" onClick={() => send(m.meta.nudge!.prompt, 'chip')}>→ {m.meta.nudge.label}</button>
                )}
              </div>
            </div>
          ))}
          {busy && (
            <div className="msg assistant">
              <span className="avatar" aria-hidden="true">म</span>
              <div className="bubble"><div className="small muted">{t('thinking')}</div><Spinner /></div>
            </div>
          )}
          <div ref={endRef} />
        </div>

        <div className="composer">
          {msgs.length === 0 && (
            <div className="suggestions">
              {suggestions.map((k) => <button key={k} className="suggestion" onClick={() => send(t(k), 'chip')}>{t(k)}</button>)}
            </div>
          )}
          {image && (
            <div className="row" style={{ marginBottom: 6 }}>
              <img src={image} alt="" style={{ width: 52, height: 52, objectFit: 'cover', borderRadius: 6, border: '1px solid var(--rule-strong)' }} />
              <button className="btn ghost sm" onClick={() => setImage('')}><Icon name="x" size={16} />{t('cancel')}</button>
            </div>
          )}
          <form className="composer-box" onSubmit={(e) => { e.preventDefault(); send(text) }}>
            <input ref={fileRef} type="file" accept="image/*" hidden onChange={async (e) => {
              const f = e.target.files?.[0]
              if (f) setImage(await fileToDataURL(f))
              e.target.value = ''
            }} />
            <button type="button" className="btn ghost icon" onClick={() => fileRef.current?.click()} aria-label={t('attach')} title={t('attach')}><Icon name="clip" /></button>
            <label htmlFor="composer" className="sr">{t('type_here')}</label>
            <textarea id="composer" ref={taRef} rows={1} value={text} placeholder={t('type_here')}
              onChange={(e) => { setText(e.target.value); e.target.style.height = 'auto'; e.target.style.height = Math.min(e.target.scrollHeight, 160) + 'px' }}
              onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(text) } }} />
            <MicButton className="btn ghost icon" onText={(s) => send(s, 'voice')} />
            <button className="btn primary icon" disabled={busy || (!text.trim() && !image)} aria-label={t('send')}><Icon name="send" /></button>
          </form>
        </div>
      </section>

      {sessions.data && sessions.data.length > 0 && (
        <aside className="stack no-print chat-rail" style={{ alignSelf: 'start', position: 'sticky', top: 70 }}>
          <Link className="btn sm" to="/app" onClick={() => { setSession(null); setMsgs([]) }}><Icon name="plus" size={16} />{t('new_chat')}</Link>
          <div className="caps muted">{t('conversations')}</div>
          <div className="stack" style={{ gap: 4 }}>
            {sessions.data.slice(0, 12).map((s) => (
              <Link key={s.id} to={`/app/c/${s.id}`} className="sheet plain" style={{ padding: '0.5rem 0.7rem', textDecoration: 'none', borderColor: s.id === session?.id ? 'var(--ink)' : undefined }}>
                <div className="small" style={{ fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{s.title}</div>
                <div className="tiny faint row between"><span>{t(`stage_${s.stage}` as Key)}</span><span className="mono">{s.updated_at.slice(5)}</span></div>
              </Link>
            ))}
          </div>
        </aside>
      )}
    </div>
  )
}
