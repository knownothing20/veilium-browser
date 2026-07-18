package evidence

import (
	"encoding/json"
	"strings"
)

func renderRunPage(nonce string, surfaces []string, origin, token string) string {
	requested, _ := json.Marshal(surfaces)
	page := runPageTemplate
	replacements := map[string]string{
		"__NONCE__":         nonce,
		"__SURFACES__":      string(requested),
		"__FRAME_URL__":     origin + "/frame/" + token,
		"__WORKER_URL__":    origin + "/worker/" + token + ".js",
		"__SUBMIT_URL__":    origin + "/submit/" + token,
		"__COMMON_SCRIPT__": browserCommonScript,
	}
	for placeholder, value := range replacements {
		page = strings.ReplaceAll(page, placeholder, value)
	}
	return page
}

func renderFramePage(nonce string, surfaces []string, origin string) string {
	requested, _ := json.Marshal(surfaces)
	page := framePageTemplate
	replacements := map[string]string{
		"__NONCE__":         nonce,
		"__SURFACES__":      string(requested),
		"__PARENT_ORIGIN__": origin,
		"__COMMON_SCRIPT__": browserCommonScript,
	}
	for placeholder, value := range replacements {
		page = strings.ReplaceAll(page, placeholder, value)
	}
	return page
}

func renderWorkerScript() string { return workerScript }

const runPageTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Veilium local evidence</title>
<style nonce="__NONCE__">
:root{font-family:system-ui,sans-serif;color-scheme:light dark}body{display:grid;place-items:center;min-height:100vh;margin:0;background:#f4f5f7;color:#1e2330}.card{width:min(520px,calc(100vw - 48px));padding:28px;border:1px solid #dfe3ea;border-radius:18px;background:#fff;box-shadow:0 18px 50px rgba(40,45,60,.1)}h1{margin:0 0 10px;font-size:22px}p{margin:0;color:#647084;line-height:1.55}.status{margin-top:20px;padding:12px 14px;border-radius:12px;background:#f0f2f6;font:13px ui-monospace,monospace}@media(prefers-color-scheme:dark){body{background:#151820;color:#eef1f7}.card{background:#20242e;border-color:#343a48}.status{background:#171a22}p{color:#aab2c1}}
</style>
</head>
<body>
<main class="card"><h1>Veilium local evidence</h1><p>This controlled loopback page observes an allowlisted browser identity surface. It does not inspect browsing data.</p><div class="status" id="status">Collecting controlled observations…</div></main>
<script nonce="__NONCE__">
const requestedSurfaces = __SURFACES__;
const frameURL = "__FRAME_URL__";
const workerURL = "__WORKER_URL__";
const submitURL = "__SUBMIT_URL__";
__COMMON_SCRIPT__

function boundedContext(label, start) {
  return new Promise((resolve) => {
    let finished = false;
    const timer = setTimeout(() => {
      if (!finished) { finished = true; resolve({snapshot:null, limitation:label + ':timeout'}); }
    }, 2500);
    start((snapshot, limitation) => {
      if (finished) return;
      finished = true;
      clearTimeout(timer);
      resolve({snapshot, limitation});
    });
  });
}

async function collectFrame() {
  return boundedContext('iframe', (done) => {
    const frame = document.createElement('iframe');
    frame.hidden = true;
    const handler = (event) => {
      if (event.origin !== location.origin || event.source !== frame.contentWindow) return;
      const data = event.data || {};
      if (data.type !== 'veilium-evidence-frame') return;
      window.removeEventListener('message', handler);
      frame.remove();
      done(data.snapshot || null, data.error ? 'iframe:' + String(data.error).slice(0,160) : '');
    };
    window.addEventListener('message', handler);
    frame.addEventListener('error', () => {
      window.removeEventListener('message', handler);
      frame.remove();
      done(null, 'iframe:load-failed');
    }, {once:true});
    frame.src = frameURL;
    document.body.appendChild(frame);
  });
}

async function collectWorker() {
  return boundedContext('worker', (done) => {
    let worker;
    try { worker = new Worker(workerURL, {name:'veilium-evidence-worker'}); }
    catch (error) { done(null, 'worker:' + safeError(error)); return; }
    worker.onmessage = (event) => {
      const data = event.data || {};
      if (data.type !== 'veilium-evidence-worker') return;
      worker.terminate();
      done(data.snapshot || null, data.error ? 'worker:' + String(data.error).slice(0,160) : '');
    };
    worker.onerror = () => { worker.terminate(); done(null, 'worker:failed'); };
    worker.postMessage({type:'collect'});
  });
}

async function runEvidence() {
  const status = document.getElementById('status');
  const limitations = [];
  const contexts = [];
  try {
    contexts.push(await collectSnapshot('top-level', requestedSurfaces));
  } catch (error) {
    status.textContent = 'Top-level collection failed.';
    throw error;
  }
  const [frameResult, workerResult] = await Promise.all([collectFrame(), collectWorker()]);
  if (frameResult.snapshot) contexts.push(frameResult.snapshot);
  if (workerResult.snapshot) contexts.push(workerResult.snapshot);
  if (frameResult.limitation) limitations.push(frameResult.limitation);
  if (workerResult.limitation) limitations.push(workerResult.limitation);
  const response = await fetch(submitURL, {
    method:'POST',
    credentials:'omit',
    cache:'no-store',
    headers:{'Content-Type':'application/json'},
    body:JSON.stringify({schemaVersion:1, contexts, limitations}),
  });
  if (!response.ok) throw new Error('submission failed with ' + response.status);
  status.textContent = 'Local evidence submitted. This page may be closed.';
}
runEvidence().catch((error) => {
  const status = document.getElementById('status');
  status.textContent = 'Evidence collection failed: ' + safeError(error);
});
</script>
</body>
</html>`

const framePageTemplate = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Veilium evidence frame</title></head>
<body>
<script nonce="__NONCE__">
const requestedSurfaces = __SURFACES__;
__COMMON_SCRIPT__
collectSnapshot('iframe', requestedSurfaces)
  .then((snapshot) => parent.postMessage({type:'veilium-evidence-frame', snapshot}, "__PARENT_ORIGIN__"))
  .catch((error) => parent.postMessage({type:'veilium-evidence-frame', error:safeError(error)}, "__PARENT_ORIGIN__"));
</script>
</body>
</html>`

const browserCommonScript = `
function safeError(error) {
  const value = error && error.message ? error.message : String(error || 'unknown');
  return value.slice(0, 160);
}
async function digestText(value) {
  const bytes = new TextEncoder().encode(String(value));
  const digest = await crypto.subtle.digest('SHA-256', bytes);
  return Array.from(new Uint8Array(digest), (item) => item.toString(16).padStart(2, '0')).join('');
}
function uaBrands() {
  const data = navigator.userAgentData;
  if (!data || !Array.isArray(data.brands)) return [];
  return data.brands.slice(0, 16).map((item) => String(item.brand).slice(0,128) + '/' + String(item.version).slice(0,32));
}
function screenSnapshot() {
  if (typeof screen === 'undefined') return null;
  return {width:screen.width,height:screen.height,availWidth:screen.availWidth,availHeight:screen.availHeight,colorDepth:screen.colorDepth,pixelDepth:screen.pixelDepth};
}
function windowSnapshot() {
  if (typeof window === 'undefined') return null;
  const viewport = window.visualViewport;
  return {outerWidth:window.outerWidth,outerHeight:window.outerHeight,innerWidth:window.innerWidth,innerHeight:window.innerHeight,viewportWidth:viewport ? viewport.width : window.innerWidth,viewportHeight:viewport ? viewport.height : window.innerHeight,viewportScale:viewport ? viewport.scale : 1,devicePixelRatio:window.devicePixelRatio || 1};
}
async function webRTCSnapshot(limitations) {
  if (typeof RTCPeerConnection !== 'function') {
    limitations.push('webrtc:unavailable');
    return {available:false,candidateTypes:[],protocols:[],usesMdns:false,gatheringState:'unavailable'};
  }
  const candidateTypes = new Set();
  const protocols = new Set();
  let usesMdns = false;
  let peer;
  try {
    peer = new RTCPeerConnection({iceServers:[]});
    peer.createDataChannel('veilium-evidence');
    peer.onicecandidate = (event) => {
      if (!event.candidate) return;
      const candidate = String(event.candidate.candidate || '');
      const typeMatch = candidate.match(/ typ ([a-z0-9]+)/i);
      const protocol = String(event.candidate.protocol || '').toLowerCase() || ((candidate.match(/\s(udp|tcp)\s/i) || [])[1] || '').toLowerCase();
      if (typeMatch) candidateTypes.add(typeMatch[1].toLowerCase());
      if (protocol === 'udp' || protocol === 'tcp') protocols.add(protocol);
      if (candidate.includes('.local')) usesMdns = true;
    };
    const offer = await peer.createOffer();
    await peer.setLocalDescription(offer);
    await Promise.race([
      new Promise((resolve) => {
        if (peer.iceGatheringState === 'complete') { resolve(); return; }
        peer.addEventListener('icegatheringstatechange', () => { if (peer.iceGatheringState === 'complete') resolve(); });
      }),
      new Promise((resolve) => setTimeout(resolve, 1200)),
    ]);
    return {available:true,candidateTypes:Array.from(candidateTypes),protocols:Array.from(protocols),usesMdns,gatheringState:String(peer.iceGatheringState || '').slice(0,64)};
  } catch (error) {
    limitations.push('webrtc:' + safeError(error));
    return {available:true,candidateTypes:Array.from(candidateTypes),protocols:Array.from(protocols),usesMdns,gatheringState:'failed'};
  } finally {
    if (peer) peer.close();
  }
}
async function canvasDigest() {
  const canvas = document.createElement('canvas'); canvas.width=240; canvas.height=80;
  const ctx = canvas.getContext('2d'); if (!ctx) throw new Error('canvas unavailable');
  ctx.fillStyle='#f4f5f7'; ctx.fillRect(0,0,240,80); ctx.font='18px sans-serif'; ctx.fillStyle='#293246'; ctx.fillText('Veilium evidence 4.2',12,32); ctx.strokeStyle='#755ee8'; ctx.beginPath(); ctx.arc(190,40,22,0,Math.PI*2); ctx.stroke();
  return digestText(canvas.toDataURL('image/png'));
}
async function webGLDigest() {
  const canvas=document.createElement('canvas'); const gl=canvas.getContext('webgl'); if(!gl) throw new Error('webgl unavailable');
  const values=[gl.getParameter(gl.VENDOR),gl.getParameter(gl.RENDERER),gl.getParameter(gl.VERSION),gl.getParameter(gl.SHADING_LANGUAGE_VERSION)];
  return digestText(JSON.stringify(values));
}
async function audioDigest() {
  if (typeof OfflineAudioContext !== 'function') throw new Error('offline audio unavailable');
  const context=new OfflineAudioContext(1,2048,44100); const oscillator=context.createOscillator(); const compressor=context.createDynamicsCompressor(); oscillator.type='triangle'; oscillator.frequency.value=1000; oscillator.connect(compressor); compressor.connect(context.destination); oscillator.start(0); const rendered=await context.startRendering(); const data=rendered.getChannelData(0); const sample=[]; for(let i=0;i<data.length;i+=64) sample.push(Number(data[i]).toFixed(8)); return digestText(sample.join(','));
}
async function clientRectsDigest() {
  const element=document.createElement('div'); element.textContent='Veilium evidence rectangle'; element.style.cssText='position:absolute;left:-9999px;top:-9999px;width:173.25px;font:15px sans-serif;line-height:1.37;padding:3.5px;border:1px solid transparent'; document.body.appendChild(element); const rect=element.getBoundingClientRect(); const value=[rect.x,rect.y,rect.width,rect.height,element.getClientRects().length].map((item)=>Number(item).toFixed(4)).join(','); element.remove(); return digestText(value);
}
async function surfaceDigests(requested, limitations) {
  if (typeof document === 'undefined') {
    for (const name of requested) limitations.push('surface:' + name + ':context-unavailable');
    return {};
  }
  const result={};
  for (const name of requested) {
    try {
      if (name==='canvas') result.canvas=await canvasDigest();
      else if (name==='webgl') result.webgl=await webGLDigest();
      else if (name==='audio') result.audio=await audioDigest();
      else if (name==='clientRects') result.clientRects=await clientRectsDigest();
    } catch (error) { limitations.push('surface:' + name + ':' + safeError(error)); }
  }
  return result;
}
async function collectSnapshot(context, requested) {
  const limitations=[];
  const timezone=Intl.DateTimeFormat().resolvedOptions().timeZone || 'unknown';
  const snapshot={context,userAgent:String(navigator.userAgent||''),uaPlatform:navigator.userAgentData ? String(navigator.userAgentData.platform||'') : '',uaBrands:uaBrands(),navigatorPlatform:String(navigator.platform||''),language:String(navigator.language||''),languages:Array.from(navigator.languages||[navigator.language]).map(String),timezone:String(timezone),hardwareConcurrency:Number(navigator.hardwareConcurrency||0),screen:screenSnapshot(),window:windowSnapshot(),webRtc:await webRTCSnapshot(limitations),surfaceDigests:await surfaceDigests(requested,limitations),limitations};
  return snapshot;
}
`

const workerScript = `
function safeError(error){const value=error&&error.message?error.message:String(error||'unknown');return value.slice(0,160)}
function uaBrands(){const data=navigator.userAgentData;if(!data||!Array.isArray(data.brands))return[];return data.brands.slice(0,16).map((item)=>String(item.brand).slice(0,128)+'/'+String(item.version).slice(0,32))}
self.onmessage=(event)=>{if(!event.data||event.data.type!=='collect')return;try{const timezone=Intl.DateTimeFormat().resolvedOptions().timeZone||'unknown';self.postMessage({type:'veilium-evidence-worker',snapshot:{context:'worker',userAgent:String(navigator.userAgent||''),uaPlatform:navigator.userAgentData?String(navigator.userAgentData.platform||''):'',uaBrands:uaBrands(),navigatorPlatform:String(navigator.platform||''),language:String(navigator.language||''),languages:Array.from(navigator.languages||[navigator.language]).map(String),timezone:String(timezone),hardwareConcurrency:Number(navigator.hardwareConcurrency||0),screen:null,window:null,webRtc:null,surfaceDigests:{},limitations:['worker:dom-and-webrtc-unavailable']}})}catch(error){self.postMessage({type:'veilium-evidence-worker',error:safeError(error)})}}
`
