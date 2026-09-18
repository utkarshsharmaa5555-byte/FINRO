import { useEffect, useState } from 'react'
import { ago, api, Link, useApp } from './lib'
import { LangSelect, ThemeToggle } from './ui'

const copy = {
  en: {
    h1a: 'Powering', h1b: 'every', h1c: 'counter.',
    lede: 'Finro catches UPI scams before they cost you, makes your shop QR, keeps your daily ledger by voice, and turns it into a real application for government funding — in Telugu, Hindi, Tamil or English.',
    start: 'Start free', demo: 'Try a demo account', see: 'See how it works',
    slip: ['12 Sep', 'Collect request “receive ₹5,000”', 'SCAM — declined', 'Sales today', '₹2,340'],
    journey_h: 'One conversation. Six pages of your ledger.',
    journey_p: 'Most apps do one thing. A merchant’s life doesn’t come in pieces — so Finro walks with you from staying safe to getting funded, remembering everything along the way.',
    steps: [
      ['Stay safe', 'Paste any SMS, call or screenshot. Finro matches it against current UPI scam patterns from RBI, NPCI and this week’s news.', 'SCAM CHECK'],
      ['Get paid right', 'A real UPI QR with your name, and a printable counter standee that warns customers about scams.', 'UPI QR'],
      ['Keep the ledger', '“Sold 40 idlis for 800, bought rice 300.” Speak it — Finro writes the ledger and tracks udhaar.', 'VOICE'],
      ['Know your business', 'Your records become a one-page blueprint with real numbers — not guesses.', 'BLUEPRINT'],
      ['Find real money', 'Schemes read live from official government portals. Every deadline is quoted from the source, or shown as “not stated”.', 'LIVE'],
      ['Apply with confidence', 'Eligibility checked rule by rule, then a draft application filled from your blueprint. Missing facts are marked, never invented.', 'DRAFT'],
    ],
    trust_h: 'Built so it can’t make things up',
    trust: [
      ['Quoted, not guessed', 'A grant deadline or amount is only shown if the exact words appear on the official page.'],
      ['Your numbers, not the model’s', 'Blueprint financials come straight from your ledger entries.'],
      ['You confirm, then it saves', 'Voice and photo entries wait for your tap before touching your ledger.'],
    ],
    live: ['schemes tracked', 'scam patterns', 'official sources', 'last checked'],
    langs_h: 'Speaks the way you do',
    demo_h: 'Demo accounts for judges & mentors',
    foot: 'Made for Synora · MSC, SRM University AP. Finro never holds your money and never asks for your PIN.',
  },
  hi: {
    h1a: 'हर गल्ले के लिए', h1b: 'एक भरोसेमंद', h1c: 'फिनरो।',
    lede: 'फिनरो UPI धोखे को नुकसान से पहले पकड़ता है, आपकी दुकान का QR बनाता है, बोलकर रोज़ का खाता रखता है, और उसे सरकारी फंडिंग के असली आवेदन में बदलता है — हिंदी, तेलुगु, तमिल या अंग्रेज़ी में।',
    start: 'मुफ़्त शुरू करें', demo: 'डेमो खाता आज़माएँ', see: 'कैसे काम करता है',
    slip: ['12 सितं', 'कलेक्ट रिक्वेस्ट “₹5,000 पाएँ”', 'धोखा — मना किया', 'आज की बिक्री', '₹2,340'],
    journey_h: 'एक बातचीत। आपके खाते के छह पन्ने।',
    journey_p: 'ज़्यादातर ऐप एक ही काम करते हैं। दुकानदार की ज़िंदगी टुकड़ों में नहीं आती — इसलिए फिनरो सुरक्षा से फंडिंग तक आपके साथ चलता है और सब याद रखता है।',
    steps: [
      ['सुरक्षित रहें', 'कोई भी SMS, कॉल या स्क्रीनशॉट डालें। फिनरो उसे RBI, NPCI और इस हफ़्ते की खबरों के UPI धोखों से मिलाता है।', 'धोखा जाँच'],
      ['सही भुगतान लें', 'आपके नाम वाला असली UPI QR, और ग्राहकों को धोखे से सावधान करने वाली प्रिंट होने लायक स्टैंडी।', 'UPI QR'],
      ['खाता रखें', '“40 इडली 800 में बेची, 300 का चावल लिया।” बस बोलिए — फिनरो खाता लिखता है और उधार का हिसाब रखता है।', 'आवाज़'],
      ['अपना व्यापार जानें', 'आपके रिकॉर्ड असली आँकड़ों वाला एक पन्ने का ब्लूप्रिंट बनते हैं — अंदाज़े नहीं।', 'ब्लूप्रिंट'],
      ['असली पैसा खोजें', 'सरकारी पोर्टल से लाइव पढ़ी गई योजनाएँ। हर अंतिम तिथि स्रोत से ली जाती है, वरना “नहीं लिखी” दिखती है।', 'लाइव'],
      ['भरोसे से आवेदन करें', 'हर नियम पर पात्रता जाँच, फिर ब्लूप्रिंट से भरा आवेदन ड्राफ्ट। कमी वाली जानकारी चिह्नित होती है, गढ़ी नहीं जाती।', 'ड्राफ्ट'],
    ],
    trust_h: 'ऐसा बना है कि बातें गढ़ न सके',
    trust: [
      ['उद्धृत, अनुमान नहीं', 'अंतिम तिथि या रकम तभी दिखती है जब वही शब्द सरकारी पेज पर हों।'],
      ['आपके आँकड़े', 'ब्लूप्रिंट के आँकड़े सीधे आपके खाते से आते हैं।'],
      ['आप पुष्टि करें, फिर सेव', 'आवाज़ और फ़ोटो वाली एंट्री आपके टैप का इंतज़ार करती हैं।'],
    ],
    live: ['योजनाएँ', 'धोखे के तरीके', 'सरकारी स्रोत', 'आख़िरी जाँच'],
    langs_h: 'आपकी भाषा में बात करता है',
    demo_h: 'जजों और मेंटरों के लिए डेमो खाते',
    foot: 'Synora के लिए बनाया · MSC, SRM University AP. फिनरो कभी आपका पैसा नहीं रखता और PIN नहीं माँगता।',
  },
  te: {
    h1a: 'ప్రతి గల్లాకు', h1b: 'నమ్మకమైన', h1c: 'ఫిన్‌రో.',
    lede: 'ఫిన్‌రో UPI మోసాలను నష్టం జరగక ముందే పట్టుకుంటుంది, మీ షాప్ QR చేస్తుంది, మాటలతో రోజువారీ ఖాతా రాస్తుంది, దాన్ని ప్రభుత్వ ఫండింగ్ కోసం నిజమైన దరఖాస్తుగా మారుస్తుంది — తెలుగు, హిందీ, తమిళం లేదా ఇంగ్లీష్‌లో.',
    start: 'ఉచితంగా మొదలుపెట్టండి', demo: 'డెమో ఖాతా ప్రయత్నించండి', see: 'ఎలా పనిచేస్తుంది',
    slip: ['12 సెప్టెం', 'కలెక్ట్ రిక్వెస్ట్ “₹5,000 పొందండి”', 'మోసం — తిరస్కరించారు', 'ఈరోజు అమ్మకం', '₹2,340'],
    journey_h: 'ఒకే సంభాషణ. మీ ఖాతాలో ఆరు పేజీలు.',
    journey_p: 'చాలా యాప్‌లు ఒకే పని చేస్తాయి. వ్యాపారి జీవితం ముక్కలుగా రాదు — అందుకే ఫిన్‌రో భద్రత నుండి ఫండింగ్ వరకు మీతో నడుస్తుంది.',
    steps: [
      ['సురక్షితంగా ఉండండి', 'ఏ SMS, కాల్ లేదా స్క్రీన్‌షాట్ అయినా పెట్టండి. RBI, NPCI, ఈ వారం వార్తల UPI మోసాలతో ఫిన్‌రో పోల్చి చూస్తుంది.', 'మోసం చెక్'],
      ['సరిగ్గా డబ్బు పొందండి', 'మీ పేరుతో నిజమైన UPI QR, కస్టమర్లను మోసాల గురించి హెచ్చరించే ప్రింట్ స్టాండీ.', 'UPI QR'],
      ['ఖాతా రాయండి', '“40 ఇడ్లీలు 800కి అమ్మాను, బియ్యం 300.” చెబితే చాలు — ఫిన్‌రో ఖాతా రాసి అప్పులు ట్రాక్ చేస్తుంది.', 'వాయిస్'],
      ['మీ వ్యాపారం తెలుసుకోండి', 'మీ రికార్డులు నిజమైన సంఖ్యలతో ఒక పేజీ బ్లూప్రింట్‌గా మారతాయి — ఊహలు కాదు.', 'బ్లూప్రింట్'],
      ['నిజమైన డబ్బు కనుగొనండి', 'అధికారిక ప్రభుత్వ పోర్టళ్ల నుండి లైవ్‌గా చదివిన పథకాలు. ప్రతి చివరి తేదీ మూలం నుండి, లేకపోతే “పేర్కొనలేదు”.', 'లైవ్'],
      ['నమ్మకంతో దరఖాస్తు చేయండి', 'ప్రతి నియమం వారీగా అర్హత చెక్, తర్వాత బ్లూప్రింట్‌తో నింపిన దరఖాస్తు. లేని వివరాలు గుర్తిస్తాం, కల్పించం.', 'డ్రాఫ్ట్'],
    ],
    trust_h: 'కల్పించలేనట్టుగా తయారు చేశాం',
    trust: [
      ['ఉల్లేఖనం, ఊహ కాదు', 'అధికారిక పేజీలో అవే మాటలు ఉంటేనే చివరి తేదీ లేదా మొత్తం చూపిస్తాం.'],
      ['మీ సంఖ్యలే', 'బ్లూప్రింట్ ఆర్థిక వివరాలు నేరుగా మీ ఖాతా నుండి.'],
      ['మీరు ఒప్పుకుంటేనే సేవ్', 'వాయిస్, ఫోటో ఎంట్రీలు మీ ట్యాప్ కోసం వేచి ఉంటాయి.'],
    ],
    live: ['పథకాలు', 'మోసపు పద్ధతులు', 'అధికారిక మూలాలు', 'చివరి చెక్'],
    langs_h: 'మీలాగే మాట్లాడుతుంది',
    demo_h: 'జడ్జీలు & మెంటర్ల కోసం డెమో ఖాతాలు',
    foot: 'Synora కోసం తయారు · MSC, SRM University AP. ఫిన్‌రో మీ డబ్బు ఎప్పుడూ ఉంచుకోదు, PIN అడగదు.',
  },
  ta: {
    h1a: 'ஒவ்வொரு கல்லாவுக்கும்', h1b: 'நம்பகமான', h1c: 'ஃபின்ரோ.',
    lede: 'ஃபின்ரோ UPI மோசடிகளை இழப்புக்கு முன்பே பிடிக்கிறது, உங்கள் கடை QR செய்கிறது, குரலால் தினசரி கணக்கு எழுதுகிறது, அதை அரசு நிதிக்கான உண்மையான விண்ணப்பமாக மாற்றுகிறது — தமிழ், தெலுங்கு, இந்தி அல்லது ஆங்கிலத்தில்.',
    start: 'இலவசமாகத் தொடங்கு', demo: 'டெமோ கணக்கை முயற்சி', see: 'எப்படி வேலை செய்கிறது',
    slip: ['12 செப்', 'கலெக்ட் கோரிக்கை “₹5,000 பெறுங்கள்”', 'மோசடி — நிராகரிக்கப்பட்டது', 'இன்றைய விற்பனை', '₹2,340'],
    journey_h: 'ஒரே உரையாடல். உங்கள் கணக்கேட்டின் ஆறு பக்கங்கள்.',
    journey_p: 'பெரும்பாலான செயலிகள் ஒரே வேலை செய்கின்றன. வியாபாரியின் வாழ்க்கை துண்டுகளாக வருவதில்லை — அதனால் ஃபின்ரோ பாதுகாப்பு முதல் நிதி வரை உங்களுடன் நடக்கிறது.',
    steps: [
      ['பாதுகாப்பாக இருங்கள்', 'எந்த SMS, அழைப்பு அல்லது ஸ்கிரீன்ஷாட்டையும் இடுங்கள். RBI, NPCI, இந்த வார செய்திகளின் UPI மோசடிகளுடன் ஒப்பிடுகிறது.', 'மோசடி சோதனை'],
      ['சரியாகப் பணம் பெறுங்கள்', 'உங்கள் பெயருடன் உண்மையான UPI QR, வாடிக்கையாளர்களை எச்சரிக்கும் அச்சிடக்கூடிய ஸ்டாண்டி.', 'UPI QR'],
      ['கணக்கு எழுதுங்கள்', '“40 இட்லி 800க்கு விற்றேன், அரிசி 300.” சொன்னால் போதும் — ஃபின்ரோ கணக்கு எழுதி கடனைக் கண்காணிக்கிறது.', 'குரல்'],
      ['உங்கள் வணிகத்தை அறியுங்கள்', 'உங்கள் பதிவுகள் உண்மையான எண்களுடன் ஒரு பக்க திட்டமாகின்றன — ஊகம் அல்ல.', 'திட்டம்'],
      ['உண்மையான நிதியைக் கண்டறியுங்கள்', 'அதிகாரப்பூர்வ அரசு தளங்களிலிருந்து நேரடியாகப் படிக்கப்பட்ட திட்டங்கள். ஒவ்வொரு கடைசி தேதியும் ஆதாரத்திலிருந்து, இல்லையெனில் “குறிப்பிடப்படவில்லை”.', 'நேரடி'],
      ['நம்பிக்கையுடன் விண்ணப்பியுங்கள்', 'ஒவ்வொரு விதிக்கும் தகுதி சோதனை, பிறகு உங்கள் திட்டத்தால் நிரப்பப்பட்ட விண்ணப்பம். இல்லாத தகவல்கள் குறிக்கப்படும், கற்பிக்கப்படாது.', 'வரைவு'],
    ],
    trust_h: 'கற்பனை செய்ய முடியாதபடி உருவாக்கப்பட்டது',
    trust: [
      ['மேற்கோள், ஊகம் அல்ல', 'அதிகாரப்பூர்வ பக்கத்தில் அதே சொற்கள் இருந்தால் மட்டுமே கடைசி தேதி அல்லது தொகை காட்டப்படும்.'],
      ['உங்கள் எண்கள்', 'திட்டத்தின் நிதி விவரங்கள் நேரடியாக உங்கள் கணக்கேட்டிலிருந்து.'],
      ['நீங்கள் உறுதிசெய்த பிறகே சேமிப்பு', 'குரல், புகைப்படப் பதிவுகள் உங்கள் தட்டலுக்காகக் காத்திருக்கும்.'],
    ],
    live: ['திட்டங்கள்', 'மோசடி முறைகள்', 'அதிகார ஆதாரங்கள்', 'கடைசி சோதனை'],
    langs_h: 'நீங்கள் பேசுவது போலவே பேசுகிறது',
    demo_h: 'நடுவர்கள் & வழிகாட்டிகளுக்கான டெமோ கணக்குகள்',
    foot: 'Synora-க்காக உருவாக்கப்பட்டது · MSC, SRM University AP. ஃபின்ரோ உங்கள் பணத்தை வைத்திருப்பதில்லை, PIN கேட்பதில்லை.',
  },
}

export const demoAccounts = [
  { name: 'Lakshmi Devi', role: 'Idli cart · Vijayawada · తెలుగు', phone: '9000000001', pin: '1111' },
  { name: 'Ramesh Gupta', role: 'Kirana store · Varanasi · हिन्दी', phone: '9000000002', pin: '2222' },
  { name: 'Mentor Desk', role: 'Admin view · English', phone: '9000000000', pin: '0000' },
]

export default function Landing() {
  const { lang, user } = useApp()
  const c = copy[lang]
  const [stats, setStats] = useState<{ grants: number; scam_patterns: number; sources: number; last_refresh: string | null } | null>(null)
  useEffect(() => { api('/public/stats').then(setStats).catch(() => {}) }, [])

  return (
    <div className="land">
      <header className="land-nav">
        <Link to="/" className="brand" style={{ padding: 0 }}>
          <span className="brand-mark">F</span>
          <span><span className="brand-name">Finro</span></span>
        </Link>
        <div className="grow" />
        <LangSelect compact />
        <ThemeToggle />
        {user ? <Link className="btn sm primary" to="/app">Finro →</Link> : <Link className="btn sm" to="/login">Log in</Link>}
      </header>

      <section className="hero">
        <div>
          <span className="stamp red" style={{ marginBottom: '1.2rem' }}>शुभ लाभ · Est. 2026</span>
          <h1 className="display">{c.h1a} <em>{c.h1b}</em> {c.h1c}</h1>
          <p className="lede">{c.lede}</p>
          <div className="row wrap" style={{ marginTop: '1.6rem' }}>
            <Link className="btn primary" to="/signup">{c.start}</Link>
            <Link className="btn" to="/login">{c.demo}</Link>
          </div>
        </div>
        <div className="book" aria-hidden="true">
          <div className="book-cover">
            <div className="swastik">
              <div className="shubh">॥ शुभ ॥</div>
              <div className="shubh" style={{ fontSize: '3.2rem' }}>मुंशी</div>
              <div className="labh">HISAAB · KITAAB</div>
            </div>
          </div>
          <div className="book-tie" />
          <div className="book-slip">
            <div className="row between"><span>{c.slip[0]}</span><span className="stamp red flat" style={{ fontSize: '0.62rem' }}>{c.slip[2]}</span></div>
            <div style={{ borderBottom: '1px dotted var(--rule-strong)', padding: '4px 0' }}>{c.slip[1]}</div>
            <div className="row between" style={{ paddingTop: 4 }}><span>{c.slip[3]}</span><strong style={{ color: 'var(--leaf)' }}>{c.slip[4]}</strong></div>
          </div>
        </div>
      </section>

      <section className="band">
        <div className="land-section" style={{ paddingTop: '2rem', paddingBottom: '2rem' }}>
          <div className="grid3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))' }}>
            <div className="counter"><div className="value">{stats?.grants ?? '—'}</div><div className="muted">{c.live[0]}</div></div>
            <div className="counter"><div className="value">{stats?.scam_patterns ?? '—'}</div><div className="muted">{c.live[1]}</div></div>
            <div className="counter"><div className="value">{stats?.sources ?? '—'}</div><div className="muted">{c.live[2]}</div></div>
            <div className="counter"><div className="value" style={{ fontSize: '1.6rem', paddingTop: '0.6rem' }}>{stats?.last_refresh ? ago(stats.last_refresh, lang) : '—'}</div><div className="muted">{c.live[3]}</div></div>
          </div>
        </div>
      </section>

      <section className="land-section" id="how">
        <h2 className="display">{c.journey_h}</h2>
        <p className="muted" style={{ maxWidth: '42rem', marginBottom: '1.5rem' }}>{c.journey_p}</p>
        <ol className="journey" style={{ padding: 0, margin: 0 }}>
          {c.steps.map(([h, p, s], i) => (
            <li key={i}>
              <span className="num">{String(i + 1).padStart(2, '0')}</span>
              <div><h3>{h}</h3><p className="muted">{p}</p></div>
              <span className={`stamp ${i === 0 ? 'red' : i >= 4 ? 'green' : ''}`}>{s}</span>
            </li>
          ))}
        </ol>
      </section>

      <section className="land-section">
        <h2 className="display">{c.trust_h}</h2>
        <div className="grid3" style={{ marginTop: '1rem' }}>
          {c.trust.map(([h, p], i) => (
            <div className="sheet" key={i}><h3>{h}</h3><p className="muted small" style={{ marginTop: 6 }}>{p}</p></div>
          ))}
        </div>
      </section>

      <section className="land-section">
        <div className="grid2">
          <div className="sheet ruled" style={{ lineHeight: '32px' }}>
            <h2 className="display" style={{ fontSize: '1.7rem', marginBottom: 8 }}>{c.langs_h}</h2>
            <div style={{ fontSize: '1.15rem' }}>
              <div>“ఈ మెసేజ్ మోసమా?”</div>
              <div>“आज 2,300 की बिक्री हुई”</div>
              <div>“என் கடைக்கு QR வேண்டும்”</div>
              <div>“Find me a loan for a second cart”</div>
            </div>
          </div>
          <div className="sheet">
            <h2 className="display" style={{ fontSize: '1.7rem', marginBottom: 10 }}>{c.demo_h}</h2>
            <table className="ledger-table">
              <tbody>
                {demoAccounts.map((d) => (
                  <tr key={d.phone}>
                    <td><strong>{d.name}</strong><div className="tiny muted">{d.role}</div></td>
                    <td className="cred amt">{d.phone}<div className="tiny muted">PIN {d.pin}</div></td>
                  </tr>
                ))}
              </tbody>
            </table>
            <Link className="btn primary sm" style={{ marginTop: 12 }} to="/login">{c.demo} →</Link>
          </div>
        </div>
      </section>

      <footer className="footer">{c.foot}</footer>
    </div>
  )
}
