package tools

const metricsAppResourceHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Metrics Preview</title>
    <style>
      :root {
        color-scheme: light;
        font-family: ui-sans-serif, system-ui, sans-serif;
      }
      body {
        margin: 0;
        background: #f4f7fb;
        color: #17324d;
      }
      .shell {
        padding: 16px;
      }
      .label {
        font-size: 12px;
        font-weight: 700;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: #55708c;
        margin: 0 0 10px;
      }
      iframe {
        width: 100%;
        min-height: 320px;
        border: 1px solid #c8d6e5;
        border-radius: 12px;
        background: #fff;
      }
    </style>
  </head>
  <body>
    <div class="shell">
      <p class="label">Metrics Preview</p>
      <iframe
        title="Metrics preview"
        srcdoc="<!doctype html><html><body style='margin:0;font-family:ui-sans-serif,system-ui,sans-serif;background:#ffffff;color:#17324d;display:grid;place-items:center;height:100vh'><div style='text-align:center'><div style='font-size:14px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;color:#55708c'>MCP App</div><div style='margin-top:12px;font-size:28px;font-weight:700'>Metrics iframe attached</div><div style='margin-top:10px;font-size:14px;color:#55708c'>Static proof-of-concept. Chart rendering will be added later.</div></div></body></html>">
      </iframe>
    </div>
  </body>
</html>
`

func MetricsAppResourceURI() string {
	return metricsAppResourceURI
}

func MetricsAppResourceMIMEType() string {
	return metricsAppResourceMimeType
}

func MetricsAppResourceHTML() string {
	return metricsAppResourceHTML
}
