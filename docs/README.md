# NeoCoin Website

This is the official NeoCoin website.

## Development

Simply open `index.html` in your browser, or serve it locally:

```bash
# Using Python
python3 -m http.server 8000

# Using Node.js
npx serve .
```

## Deployment to GitHub Pages

1. Go to Repository Settings → Pages
2. Source: Deploy from a branch
3. Branch: main, folder: /docs
4. Save

## Custom Domain

To use neocoin.io:
1. Buy domain from Namecheap/Cloudflare
2. Add CNAME record: @ → yourusername.github.io
3. Add CNAME file to /docs folder with content: neocoin.io

## Tech Stack

- Pure HTML/CSS/JS
- No frameworks required
- Mobile responsive
- Dark theme
