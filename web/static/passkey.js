async function canPasskey() {
  if (!window.PublicKeyCredential) {
    return false;
  }

  return window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
}

async function createPasskey() {
  const { id, challenge } = await fetch('/passkey/register/start').then(resp => resp.json());

  let credential = await navigator.credentials.create({
    publicKey: {
      // Unused in this example.
      challenge: Uint8Array.from(atob(challenge), c => c.charCodeAt(0)),
      pubKeyCredParams: [
        { type: "public-key", alg: -7, },
        { type: "public-key", alg: -257 },
      ],
      rp: { name: window.location.host },
      user: {
        id: Uint8Array.from(atob(id), c => c.charCodeAt(0)),
        name: "http://me.example.com", // TODO: inject this value
        displayName: ""
      },
    }
  });

  await fetch('/passkey/register/end', {
    method: 'post',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credential),
  });
}

canPasskey()
  .then(ok => {
    if (ok) {
      const createPasskeyButton = document.getElementById('createPasskey');
      createPasskeyButton.onclick = createPasskey;

      const verifyPasskeyButton = document.getElementById('verifyPasskey');
      verifyPasskeyButton.onclick = verifyPasskey;
    }
  })
  .catch(err => console.warn(err));
