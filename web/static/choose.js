const urlParams = new URLSearchParams(window.location.search);

const methods = document.querySelector('.methods');
const info = document.querySelector('.info');
const cachedAt = document.querySelector('.cachedAt');
const refresh = document.getElementById('refresh');
const loader = document.querySelector('.loader');

var socket = new WebSocket(`${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`);
socket.onopen = function(event) {
  socket.send(JSON.stringify({
    me: urlParams.get('me'),
    clientID: urlParams.get('client_id'),
    redirectURI: urlParams.get('redirect_uri'),
    force: false,
  }));
};

refresh.onclick = function() {
  while (methods.firstChild) {
    methods.removeChild(methods.firstChild);
  }
  info.classList.add('loading');
  loader.classList.remove('hide');

  socket.send(JSON.stringify({
    me: urlParams.get('me'),
    clientID: urlParams.get('client_id'),
    redirectURI: urlParams.get('redirect_uri'),
    force: true,
  }));
};

const elements = {};
let anyVerified = false;

socket.onmessage = function(event) {
  const profile = JSON.parse(event.data);

  if (profile.Type) {
    switch (profile.Type) {
      case 'error':
        if (profile.Link === '') {
          showError(methods);
          return;
        } else {
          toError(elements[profile.Link], 'error');
        }
        break;
      case 'pgp':
        const pgpEl = renderText(methods, profile.Link);
        toMethod(pgpEl, profile.Method);
        anyVerified = true;
        break;
      case 'passkey':
        canPasskey()
          .then(ok => {
            if (ok) {
              const passkeyEl = renderText(methods, profile.Link);
              toPasskey(passkeyEl, profile.Method);
              anyVerified = true;
            }
          }).
          catch(err => console.warn(err));
        break;
      case 'found':
        elements[profile.Link] = renderText(methods, profile.Link);
        break;
      case 'not-supported':
        toError(elements[profile.Link], 'unsupported');
        break;
      case 'unverified':
        methods.removeChild(elements[profile.Link]);
        break;
      case 'verified':
        elements[profile.Link] = toMethod(elements[profile.Link], profile.Method);
        anyVerified = true;
        break;
      case 'done':
        loader.classList.add('hide');
        if (!anyVerified) {
          const errorText = document.createTextNode("Sorry, you aren't able to use any of the supported authentication providers.");
          methods.classList.add('info');
          methods.appendChild(errorText);
        }
        break;
    }
  } else {
    loader.classList.add('hide');

    cachedAt.textContent = profile.CachedAt;
    info.classList.remove('loading');

    if (!profile.Methods || profile.Methods.length === 0) {
      const errorText = document.createTextNode("Sorry, you aren't able to use any of the supported authentication providers.");
      methods.classList.add('info');
      methods.appendChild(errorText);
    } else {
      for (const method of profile.Methods) {
        renderMethod(methods, method);
      }
    }
  }
}

function showError(root) {
  const p = document.createElement('p');
  const text = document.createTextNode('Something went wrong while trying to retrieve possible authentication methods.');

  p.appendChild(text);
  p.classList.add('error-msg');
  root.appendChild(p);
  loader.classList.add('hide');
}

function renderMethod(root, method) {
  const li = document.createElement('li');

  const btn = document.createElement('a');
  btn.classList.add('btn');
  btn.href = '/auth/start?' + method.Query;

  const name = document.createElement('strong');
  name.textContent = method.StrategyName;

  const asText = document.createTextNode(' as ' + method.ProfileURL);

  btn.appendChild(name);
  btn.appendChild(asText);
  li.appendChild(btn);
  root.appendChild(li);
  return li;
}

function renderText(root, link) {
  const li = document.createElement('li');
  const text = document.createTextNode(link);

  li.appendChild(text);
  root.appendChild(li);
  return li;
}

function toError(li, errorClass) {
  const errorText = errorClass === 'unsupported'
    ? document.createTextNode(' is not supported for authentication')
    : document.createTextNode(' could not be retrieved');

  li.classList.add(errorClass);
  li.appendChild(errorText);
}

function toMethod(li, method) {
  while (li.firstChild) {
    li.removeChild(li.firstChild);
  }

  const btn = document.createElement('a');
  btn.classList.add('btn');
  btn.href = '/auth/start?' + method.Query;

  const name = document.createElement('strong');
  name.textContent = method.StrategyName;

  const asText = document.createTextNode(' as ' + method.ProfileURL);

  btn.appendChild(name);
  btn.appendChild(asText);
  li.appendChild(btn);
  return li;
}

function toPasskey(li, method) {
  while (li.firstChild) {
    li.removeChild(li.firstChild);
  }

  const btn = document.createElement('a');
  btn.classList.add('btn');
  btn.href = '#';
  btn.onclick = verifyPasskey;

  const name = document.createElement('strong');
  name.textContent = method.StrategyName;

  btn.appendChild(name);
  li.appendChild(btn);
  return li;
}

async function canPasskey() {
  if (!window.PublicKeyCredential) {
    return false;
  }

  return window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
}

async function verifyPasskey() {
  const { credentialID, challenge } = await fetch('/passkey/authorize').then(resp => resp.json());

  let credential = await navigator.credentials
    .get({
      publicKey: {
        allowCredentials: [{
          type: 'public-key',
          id: Uint8Array.from(atob(credentialID), c => c.charCodeAt(0)),
        }],
        challenge: Uint8Array.from(atob(challenge), c => c.charCodeAt(0)),
      },
    });

  const resp = await fetch('/passkey/authorize', {
    method: 'post',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credential),
  });

  if (resp.status === 200) {
    const { redirectURI } = await resp.json();

    window.location.replace(redirectURI);
  }
}
