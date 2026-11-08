import { mount } from 'svelte';
import './app.css';
import App from './App.svelte';
import { boot, installRouter, readInitial } from './lib/nav.svelte';
import { asset } from './lib/asset';

const target = document.getElementById('app')!;
const initial = readInitial();

installRouter();
void boot(initial).then(() => {
  // Replace the server-rendered HTML with the live app in one task (no paint between).
  target.textContent = '';
  mount(App, { target });
});

// Use the illustrated favicon (images/art/favicon.png) once it exists; until
// then the inline SVG in index.html stays.
const fav = new Image();
fav.onload = () => {
  const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (link) {
    link.type = 'image/png';
    link.href = fav.src;
  }
};
fav.src = asset('/images/art/favicon.png');
