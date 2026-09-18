import { useState } from 'react'
import { api, fileToDataURL, useApp, useData } from '../lib'
import { ErrorLine, Icon, MicButton, PageHead, PatternCard, QRCard, Spinner, VerdictCard, type Pattern, type QR, type Verdict } from '../ui'

export function Fraud() {
  const { t, lang } = useApp()
  const patterns = useData<{ catalogue: Pattern[]; news: Pattern[] }>('/fraud/patterns', [lang])
  const [text, setText] = useState('')
  const [image, setImage] = useState('')
  const [verdict, setVerdict] = useState<Verdict | null>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const check = async () => {
    setBusy(true); setErr(''); setVerdict(null)
    try { setVerdict(await api<Verdict>('/fraud/check', { body: { text, image } })) } catch (e) { setErr((e as Error).message) } finally { setBusy(false) }
  }

  return (
    <>
      <PageHead kicker={t('fraud_kicker')} title={t('fraud_title')}>
        <a className="btn red sm" href="tel:1930">☎ 1930</a>
      </PageHead>
      <div className="grid2" style={{ alignItems: 'start' }}>
        <form className="sheet stack" onSubmit={(e) => { e.preventDefault(); check() }}>
          <h3>{t('check_msg')}</h3>
          <label className="sr" htmlFor="scamtext">{t('paste_msg')}</label>
          <textarea id="scamtext" className="textarea" rows={5} value={text} onChange={(e) => setText(e.target.value)} placeholder={t('paste_msg')} />
          {image && <div className="row"><img src={image} alt="" style={{ width: 64, height: 64, objectFit: 'cover', borderRadius: 6 }} /><button type="button" className="btn ghost sm" onClick={() => setImage('')}>{t('cancel')}</button></div>}
          <div className="row wrap">
            <label className="btn sm">
              <Icon name="camera" size={16} />{t('attach')}
              <input type="file" accept="image/*" hidden onChange={async (e) => { const f = e.target.files?.[0]; if (f) setImage(await fileToDataURL(f)); e.target.value = '' }} />
            </label>
            <MicButton className="btn sm icon" onText={(s) => setText((x) => (x ? x + ' ' : '') + s)} />
            <div className="grow" />
            <button className="btn primary" disabled={busy || (!text.trim() && !image)}>{busy ? <Spinner /> : <Icon name="shield" size={18} />}{t('check')}</button>
          </div>
          <ErrorLine msg={err} />
        </form>
        <div>{verdict ? <VerdictCard v={verdict} /> : (
          <div className="sheet ruled" style={{ lineHeight: '32px' }}>
            <span className="stamp red">{t('helpline')}</span>
            <p style={{ marginTop: 8 }}>cybercrime.gov.in · Sanchar Saathi (Chakshu)</p>
          </div>
        )}</div>
      </div>

      {patterns.data && patterns.data.news.length > 0 && (
        <section style={{ marginTop: '2rem' }}>
          <h2 className="display" style={{ fontSize: '1.5rem', marginBottom: '0.6rem' }}>{t('in_news')}</h2>
          <div className="sheet">
            <table className="ledger-table"><tbody>
              {patterns.data.news.map((n) => (
                <tr key={n.id}>
                  <td className="mono tiny faint" style={{ width: 90 }}>{n.published_at}</td>
                  <td><a href={n.source_url} target="_blank" rel="noopener noreferrer">{n.title}</a><div className="small muted">{n.description}</div></td>
                  <td><span className="chip red">{n.pattern_slug}</span></td>
                </tr>
              ))}
            </tbody></table>
          </div>
        </section>
      )}

      <section style={{ marginTop: '2rem' }}>
        <h2 className="display" style={{ fontSize: '1.5rem', marginBottom: '0.6rem' }}>{t('catalogue')}</h2>
        {patterns.loading && <Spinner />}
        <div className="grid2">{patterns.data?.catalogue.map((p) => <PatternCard key={p.id} p={p} />)}</div>
      </section>
    </>
  )
}

export function QRPage() {
  const { t } = useApp()
  const list = useData<QR[]>('/qr')
  const profile = useData<{ profile: any }>('/profile')
  const [vpa, setVpa] = useState('')
  const [payee, setPayee] = useState('')
  const [amount, setAmount] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  const p = profile.data?.profile

  const make = async () => {
    const numAmt = Number(amount)
    if (!amount.trim() || isNaN(numAmt) || numAmt <= 0) {
      setErr(t('err_amount'))
      return
    }
    setBusy(true); setErr('')
    try {
      await api('/qr', { body: { vpa: vpa || p?.upi_vpa, payee: payee || p?.business_name, amount_paise: Math.round(numAmt * 100) } })
      setVpa(''); setPayee(''); setAmount('')
      list.reload()
    } catch (e) { setErr((e as Error).message) } finally { setBusy(false) }
  }

  return (
    <>
      <PageHead kicker={t('qr_kicker')} title={t('qr_title')} />
      <div className="grid2" style={{ alignItems: 'start' }}>
        <form className="sheet stack" onSubmit={(e) => { e.preventDefault(); make() }}>
          <div className="field"><label htmlFor="vpa">{t('vpa')}</label><input id="vpa" className="input mono" placeholder={p?.upi_vpa || 'name@okaxis'} value={vpa} onChange={(e) => setVpa(e.target.value.trim())} autoCapitalize="off" /></div>
          <div className="field"><label htmlFor="payee">{t('payee')}</label><input id="payee" className="input" placeholder={p?.business_name || ''} value={payee} onChange={(e) => setPayee(e.target.value)} maxLength={60} /></div>
          <div className="field">
            <label htmlFor="amt">{t('amount_opt')} <span style={{ color: 'var(--stamp)' }}>*</span></label>
            <input id="amt" className="input mono" inputMode="decimal" value={amount} onChange={(e) => setAmount(e.target.value.replace(/[^\d.]/g, ''))} placeholder="₹" required />
          </div>
          <ErrorLine msg={err} />
          <button className="btn primary" disabled={busy || !amount.trim() || Number(amount) <= 0}>
            {busy ? <Spinner /> : <Icon name="qr" size={18} />}{t('make_qr')}
          </button>
        </form>
        <div className="stack">
          {list.loading && <Spinner />}
          {list.data?.slice(0, 3).map((q) => <QRCard key={q.id} q={q} />)}
        </div>
      </div>
    </>
  )
}

const standeeCopy: Record<string, { scan: string; tip: string }> = {
  en: { scan: 'Scan & pay with any UPI app', tip: 'We will never ask you to scan a QR to receive money, or to share your UPI PIN.' },
  hi: { scan: 'किसी भी UPI ऐप से स्कैन करके भुगतान करें', tip: 'हम कभी पैसे पाने के लिए QR स्कैन करने या UPI PIN बताने को नहीं कहेंगे।' },
  te: { scan: 'ఏ UPI యాప్‌తోనైనా స్కాన్ చేసి చెల్లించండి', tip: 'డబ్బు పొందడానికి QR స్కాన్ చేయమని లేదా UPI PIN చెప్పమని మేము ఎప్పుడూ అడగము.' },
  ta: { scan: 'எந்த UPI செயலியிலும் ஸ்கேன் செய்து செலுத்துங்கள்', tip: 'பணம் பெற QR ஸ்கேன் செய்யவோ UPI PIN சொல்லவோ நாங்கள் ஒருபோதும் கேட்க மாட்டோம்.' },
}

export function Standee({ id }: { id: number }) {
  const { t, lang } = useApp()
  const list = useData<QR[]>('/qr')
  const q = list.data?.find((x) => x.id === id)
  const [busy, setBusy] = useState(false)
  if (list.loading) return <Spinner />
  if (!q) return <p>QR not found</p>
  const c = standeeCopy[lang]

  const png = async () => {
    setBusy(true)
    try {
      const W = 1200, H = 1700
      const cv = document.createElement('canvas')
      cv.width = W; cv.height = H
      const g = cv.getContext('2d')!
      await document.fonts.ready
      g.fillStyle = '#fbf7ee'; g.fillRect(0, 0, W, H)
      g.fillStyle = '#8e2a1e'; g.fillRect(0, 0, W, 300)
      g.strokeStyle = '#e9c46a'; g.setLineDash([22, 14]); g.lineWidth = 8; g.beginPath(); g.moveTo(0, 300); g.lineTo(W, 300); g.stroke(); g.setLineDash([])
      g.textAlign = 'center'
      g.fillStyle = '#f6e7c8'; g.font = '110px "Rozha One", serif'; g.fillText(q.payee, W / 2, 190, W - 120)
      const img = new Image()
      img.src = `/api/qr/${q.id}/png?size=1024`
      await img.decode()
      g.imageSmoothingEnabled = false
      g.drawImage(img, 150, 380, 900, 900)
      g.fillStyle = '#1f2a44'; g.font = '60px "Courier Prime", monospace'; g.fillText(q.vpa, W / 2, 1360)
      g.font = '600 50px "Hind", "Noto Sans Telugu", "Noto Sans Tamil", sans-serif'; g.fillText(c.scan, W / 2, 1450, W - 100)
      g.fillStyle = '#f5dcd2'; g.fillRect(80, 1500, W - 160, 150)
      g.fillStyle = '#8e2a1e'; g.font = '600 38px "Hind", "Noto Sans Telugu", "Noto Sans Tamil", sans-serif'
      wrap(g, c.tip, W / 2, 1565, W - 220, 48)
      const a = document.createElement('a')
      a.href = cv.toDataURL('image/png')
      a.download = `finro-standee-${q.vpa}.png`
      a.click()
    } finally { setBusy(false) }
  }

  return (
    <>
      <div className="row between wrap no-print" style={{ marginBottom: '1rem' }}>
        <button className="btn ghost sm" onClick={() => history.back()}>← {t('cancel')}</button>
        <div className="row">
          <button className="btn sm" onClick={png} disabled={busy}>{busy ? <Spinner /> : <Icon name="download" size={16} />}{t('download_png')}</button>
          <button className="btn primary sm" onClick={() => print()}><Icon name="printer" size={16} />{t('export_pdf')}</button>
        </div>
      </div>
      <div className="standee">
        <div className="standee-top"><div className="display" style={{ fontSize: '2rem' }}>{q.payee}</div></div>
        <div className="standee-body">
          <img src={`/api/qr/${q.id}/png?size=720`} alt={`UPI QR for ${q.payee}`} />
          <div className="mono" style={{ fontSize: '1.1rem', fontWeight: 700 }}>{q.vpa}</div>
          <div style={{ fontWeight: 600 }}>{c.scan}</div>
          <div className="standee-tip">🛡 {c.tip}</div>
          <div className="tiny" style={{ color: '#7d8497' }}>Finro · UPI</div>
        </div>
      </div>
    </>
  )
}

function wrap(g: CanvasRenderingContext2D, text: string, x: number, y: number, max: number, lh: number) {
  const words = text.split(' ')
  let line = ''
  for (const w of words) {
    const test = line ? line + ' ' + w : w
    if (g.measureText(test).width > max && line) { g.fillText(line, x, y); line = w; y += lh } else line = test
  }
  g.fillText(line, x, y)
}
