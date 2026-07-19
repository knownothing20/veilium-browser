package networkevidence

type browserProbeConfig struct {
	SchemaVersion int                      `json:"schemaVersion"`
	SubmitURL     string                   `json:"submitUrl"`
	DNSToken      string                   `json:"dnsToken"`
	Definitions   []browserProbeDefinition `json:"definitions"`
}

type browserProbeDefinition struct {
	ID               string    `json:"id"`
	Revision         int       `json:"revision"`
	Kind             ProbeKind `json:"kind"`
	HTTPSURL         string    `json:"httpsUrl,omitempty"`
	STUNServer       string    `json:"stunServer,omitempty"`
	DNSZone          string    `json:"dnsZone,omitempty"`
	DNSResultURL     string    `json:"dnsResultUrl,omitempty"`
	TimeoutMS        int       `json:"timeoutMs"`
	MaxResponseBytes int       `json:"maxResponseBytes,omitempty"`
}

func browserProbeDefinitions(set ProbeSet) []browserProbeDefinition {
	definitions := make([]browserProbeDefinition, 0, len(set.Definitions))
	for _, definition := range set.Definitions {
		definitions = append(definitions, browserProbeDefinition{
			ID:               definition.ID,
			Revision:         definition.Revision,
			Kind:             definition.Kind,
			HTTPSURL:         definition.HTTPSURL,
			STUNServer:       definition.STUNServer,
			DNSZone:          definition.DNSZone,
			DNSResultURL:     definition.DNSResultURL,
			TimeoutMS:        definition.TimeoutSeconds * 1000,
			MaxResponseBytes: definition.MaxResponseBytes,
		})
	}
	return definitions
}

const browserProbeScript = `'use strict';
const statusNode=document.getElementById('status');
const clean=(value,limit=512)=>String(value??'').trim().slice(0,limit);
const unique=(values)=>Array.from(new Set(values.map((value)=>clean(value,256)).filter(Boolean))).sort();
const observation=(definition,status,values=[],reasonCode='',detail='')=>({probeId:definition.id,probeRevision:definition.revision,probeKind:definition.kind,status,values:unique(values),reasonCode:clean(reasonCode,128),detail:clean(detail,1024)});
const sleep=(milliseconds)=>new Promise((resolve)=>setTimeout(resolve,milliseconds));

function configURL(){
  const token=location.pathname.split('/').filter(Boolean).pop();
  return '/config/'+token+'.json';
}

async function boundedJSON(rawURL,definition){
  const controller=new AbortController();
  const timer=setTimeout(()=>controller.abort(),definition.timeoutMs);
  try{
    const response=await fetch(rawURL,{cache:'no-store',credentials:'omit',redirect:'error',referrerPolicy:'no-referrer',signal:controller.signal});
    if(!response.ok) throw new Error('probe returned HTTP '+response.status);
    const text=await response.text();
    const size=new TextEncoder().encode(text).length;
    if(definition.maxResponseBytes>0&&size>definition.maxResponseBytes) throw new Error('probe response exceeded configured limit');
    return JSON.parse(text);
  }finally{clearTimeout(timer);}
}

async function runExitIP(definition){
  try{
    const data=await boundedJSON(definition.httpsUrl,definition);
    if(!data||typeof data.ip!=='string'||!data.ip.trim()) throw new Error('probe response did not contain an IP string');
    return observation(definition,'passed',[data.ip]);
  }catch(error){
    return observation(definition,'unavailable',[],'exit-ip-unavailable',error instanceof Error?error.message:String(error));
  }
}

function parseCandidate(raw,values){
  const fields=String(raw||'').trim().split(/\s+/);
  const typeIndex=fields.indexOf('typ');
  if(fields.length<8||typeIndex<0||typeIndex+1>=fields.length) return;
  const protocol=clean(fields[2],16).toLowerCase();
  const address=clean(fields[4],128);
  const candidateType=clean(fields[typeIndex+1],16).toLowerCase();
  if(['host','srflx','prflx','relay'].includes(candidateType)) values.add('candidate:'+candidateType);
  if(['udp','tcp'].includes(protocol)) values.add('protocol:'+protocol);
  if(candidateType==='host'&&address.endsWith('.local')) values.add('mdns:true');
  if(candidateType==='host'&&!address.endsWith('.local')) values.add('mdns:false');
  if(['srflx','prflx','relay'].includes(candidateType)&&address) values.add('public-ip:'+address);
}

async function runSTUN(definition){
  if(typeof RTCPeerConnection!=='function') return observation(definition,'unavailable',[],'webrtc-unavailable','RTCPeerConnection is unavailable');
  const values=new Set();
  const connection=new RTCPeerConnection({iceServers:[{urls:[definition.stunServer]}],iceCandidatePoolSize:0});
  try{
    connection.createDataChannel('veilium-network-evidence');
    const finished=new Promise((resolve)=>{
      const timer=setTimeout(resolve,definition.timeoutMs);
      connection.onicecandidate=(event)=>{
        if(event.candidate){parseCandidate(event.candidate.candidate,values);return;}
        clearTimeout(timer);resolve();
      };
    });
    const offer=await connection.createOffer({offerToReceiveAudio:false,offerToReceiveVideo:false});
    await connection.setLocalDescription(offer);
    await finished;
    const result=Array.from(values);
    if(result.length===0) return observation(definition,'unavailable',[],'stun-no-candidates','No bounded ICE candidate summary was observed');
    const publicAddress=result.some((value)=>value.startsWith('public-ip:'));
    return observation(definition,publicAddress?'passed':'partial',result,publicAddress?'':'stun-no-public-address',publicAddress?'':'ICE completed without a reflexive or relay address');
  }catch(error){
    return observation(definition,'unavailable',[],'stun-unavailable',error instanceof Error?error.message:String(error));
  }finally{connection.close();}
}

async function runDNS(definition,dnsToken){
  const zone=String(definition.dnsZone||'').replace(/\.$/,'');
  const trigger='https://'+dnsToken+'.'+zone+'/veilium-network-evidence';
  try{
    try{await fetch(trigger,{mode:'no-cors',cache:'no-store',credentials:'omit',redirect:'error',referrerPolicy:'no-referrer'});}catch(_error){}
    await sleep(Math.min(1000,Math.max(200,Math.floor(definition.timeoutMs/4))));
    const resultURL=new URL(definition.dnsResultUrl);
    resultURL.searchParams.set('token',dnsToken);
    const data=await boundedJSON(resultURL.toString(),definition);
    const seen=Boolean(data&&data.seen===true);
    const values=['seen:'+(seen?'true':'false')];
    if(data&&typeof data.resolverIp==='string'&&data.resolverIp.trim()) values.push('resolver-ip:'+data.resolverIp);
    if(data&&typeof data.rcode==='string'&&data.rcode.trim()) values.push('rcode:'+data.rcode.toUpperCase());
    return observation(definition,seen?'passed':'partial',values,seen?'':'dns-query-not-seen',seen?'':'The delegated query was not observed before the result deadline');
  }catch(error){
    return observation(definition,'unavailable',[],'dns-probe-unavailable',error instanceof Error?error.message:String(error));
  }
}

async function runDefinition(definition,dnsToken){
  if(definition.kind==='exit-ip') return runExitIP(definition);
  if(definition.kind==='webrtc-stun') return runSTUN(definition);
  if(definition.kind==='delegated-dns') return runDNS(definition,dnsToken);
  return observation(definition,'skipped',[],'unsupported-probe-kind','The controlled page did not recognize this probe kind');
}

(async()=>{
  const limitations=[];
  try{
    const config=await boundedJSON(configURL(),{timeoutMs:5000,maxResponseBytes:65536});
    const observations=await Promise.all(config.definitions.map((definition)=>runDefinition(definition,config.dnsToken)));
    const response=await fetch(config.submitUrl,{method:'POST',cache:'no-store',credentials:'omit',redirect:'error',referrerPolicy:'no-referrer',headers:{'Content-Type':'application/json'},body:JSON.stringify({schemaVersion:config.schemaVersion,observations,limitations})});
    if(!response.ok) throw new Error('collector returned HTTP '+response.status);
    statusNode.textContent='Network evidence submitted.';
  }catch(error){
    statusNode.textContent='Network evidence failed: '+clean(error instanceof Error?error.message:String(error));
  }
})();`
