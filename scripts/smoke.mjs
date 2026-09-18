// End-to-end smoke test against a deployed Finro. Usage: node scripts/smoke.mjs [baseURL]
const BASE = process.argv[2] || 'https://munshi-ochre.vercel.app'
let cookie = ''
const results = []

async function call(method, path, body) {
  const t0 = Date.now()
  const res = await fetch(BASE + '/api' + path, {
    method,
    headers: { 'Content-Type': 'application/json', cookie },
    body: body ? JSON.stringify(body) : undefined,
  })
  const sc = res.headers.get('set-cookie')
  if (sc) cookie = sc.split(';')[0]
  const ct = res.headers.get('content-type') || ''
  const data = ct.includes('json') ? await res.json() : await res.arrayBuffer()
  return { status: res.status, data, ms: Date.now() - t0 }
}

async function check(name, fn) {
  try {
    const detail = await fn()
    results.push(['PASS', name, detail])
  } catch (e) {
    results.push(['FAIL', name, e.message])
  }
}
const assert = (cond, msg) => { if (!cond) throw new Error(msg) }

await check('login rejects wrong PIN', async () => {
  const r = await call('POST', '/auth/login', { phone: '9000000002', pin: '9999' })
  assert(r.status === 401, 'expected 401, got ' + r.status)
})
await check('login Ramesh', async () => {
  const r = await call('POST', '/auth/login', { phone: '9000000002', pin: '2222' })
  assert(r.status === 200 && r.data.lang === 'hi', JSON.stringify(r.data))
})
await check('khata summary has seeded numbers', async () => {
  const r = await call('GET', '/records?days=30')
  assert(r.data.summary.sales_paise > 0 && r.data.entries.length > 0, 'no records')
  return `sales ₹${r.data.summary.sales_paise / 100}, ${r.ms}ms`
})
await check('khata voice parse (Hindi)', async () => {
  const r = await call('POST', '/records/parse', { text: 'आज 40 पैकेट नमकीन 1200 रुपये में बेचे और 500 का तेल खरीदा, शर्मा जी ने 300 उधार लिए', source: 'voice' })
  assert(r.status === 200 && r.data.entries.length >= 3, JSON.stringify(r.data))
  const kinds = r.data.entries.map((e) => `${e.kind}:${e.amount_paise / 100}`).join(', ')
  assert(r.data.entries.some((e) => e.kind === 'sale' && e.amount_paise === 120000), 'sale 1200 missing: ' + kinds)
  return `${kinds} (${r.ms}ms)`
})
await check('QR validates bad UPI ID', async () => {
  const r = await call('POST', '/qr', { vpa: 'not-a-vpa', payee: 'Shop', amount_paise: 50000 })
  assert(r.status === 400, 'expected 400, got ' + r.status)
})
await check('QR rejects missing or zero amount', async () => {
  const r = await call('POST', '/qr', { vpa: 'guptakirana@ybl', payee: 'Gupta General Store', amount_paise: 0 })
  assert(r.status === 400, 'expected 400, got ' + r.status)
})
await check('QR create + PNG', async () => {
  const r = await call('POST', '/qr', { vpa: 'guptakirana@ybl', payee: 'Gupta General Store', amount_paise: 50000 })
  assert(r.status === 200 && r.data.upi_link.startsWith('upi://pay?'), JSON.stringify(r.data))
  const png = await call('GET', `/qr/${r.data.id}/png?size=300`)
  assert(png.status === 200 && png.data.byteLength > 500, 'png failed')
  return r.data.upi_link
})
await check('fraud check flags collect-request scam', async () => {
  const r = await call('POST', '/fraud/check', { text: 'Dear customer, you have received a payment request of Rs 4999 from PAYTM-CASHBACK. Enter UPI PIN to receive the cashback now.' })
  assert(r.status === 200 && r.data.verdict === 'scam', JSON.stringify(r.data))
  return `${r.data.verdict} ${r.data.confidence}% [${r.data.matched_patterns}] ${r.ms}ms`
})
await check('grants list has verified sources + no fabricated deadlines', async () => {
  const r = await call('GET', '/grants')
  assert(r.data.grants.length > 10, 'few grants')
  const withDeadline = r.data.grants.filter((g) => g.deadline)
  assert(withDeadline.every((g) => g.deadline_raw), 'deadline without source quote')
  return `${r.data.grants.length} grants, ${withDeadline.length} with quoted deadlines, ${r.data.sources.length} sources`
})
await check('blueprint uses khata numbers', async () => {
  const r = await call('POST', '/blueprint', { notes: 'Wants to launch packaged namkeen brand Kashi Crunch' })
  assert(r.status === 200, JSON.stringify(r.data))
  const s = await call('GET', '/records?days=30')
  assert(r.data.data.financials.sales_last_30d_inr === Math.floor(s.data.summary.sales_paise / 100), 'financials drifted')
  return `${r.data.data.one_liner} (${r.ms}ms)`
})
let fits
await check('matching scores rule by rule', async () => {
  const r = await call('POST', '/matches', {})
  assert(r.status === 200 && r.data.fits.length > 0, JSON.stringify(r.data).slice(0, 300))
  fits = r.data.fits
  const biotech = fits.find((f) => f.grant?.source === 'BIRAC')
  return `${fits.length} scored, top: ${fits[0].grant.title} ${fits[0].score}; BIRAC verdict: ${biotech?.verdict ?? 'excluded'} (${r.ms}ms)`
})
await check('draft application with NEEDED placeholders', async () => {
  const best = fits?.find((f) => f.verdict !== 'not_a_fit')
  assert(best, 'no fitting scheme')
  const r = await call('POST', '/drafts', { grant_id: best.grant_id })
  assert(r.status === 200 && r.data.sections.length >= 4, JSON.stringify(r.data).slice(0, 300))
  return `${r.data.title}: ${r.data.sections.length} sections, ${r.data.needed_count} NEEDED (${r.ms}ms)`
})
await check('help desk answers from FAQ', async () => {
  const r = await call('POST', '/help/ask', { question: 'Do I need my PIN to receive money?' })
  assert(r.status === 200 && r.data.answer, JSON.stringify(r.data))
  return r.data.answer.slice(0, 80)
})
await check('TTS returns audio', async () => {
  const r = await call('POST', '/voice/tts', { text: 'नमस्ते रमेश जी' })
  if (r.status === 502) return 'server voice unavailable for this key → browser speech fallback in UI'
  assert(r.status === 200 && r.data.byteLength > 1000, 'status ' + r.status)
  return `${r.data.byteLength} bytes, ${r.ms}ms`
})
await check('non-admin blocked from admin', async () => {
  const r = await call('GET', '/admin/overview')
  assert(r.status === 403, 'expected 403, got ' + r.status)
})
await check('admin overview', async () => {
  await call('POST', '/auth/login', { phone: '9000000000', pin: '0000' })
  const r = await call('GET', '/admin/overview')
  assert(r.status === 200 && r.data.counts.merchants >= 2, JSON.stringify(r.data).slice(0, 200))
  return JSON.stringify(r.data.counts)
})

for (const [s, n, d] of results) console.log(`${s}  ${n}${d ? '  →  ' + d : ''}`)
process.exit(results.some((r) => r[0] === 'FAIL') ? 1 : 0)
