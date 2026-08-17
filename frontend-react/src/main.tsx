import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './styles.css';
import { App } from './app/App';

// The retired Angular frontend registered a service worker on this origin.
// Remove it and its caches once the React shell is reached so it cannot keep
// serving the old application after a deployment.
if ('serviceWorker' in navigator) {
  void navigator.serviceWorker.getRegistrations().then(async (registrations) => {
    await Promise.all(registrations.map((registration) => registration.unregister()));

    if ('caches' in globalThis) {
      const cacheNames = await caches.keys();
      await Promise.all(cacheNames.map((cacheName) => caches.delete(cacheName)));
    }
  });
}

createRoot(document.getElementById('root')!).render(<StrictMode><App /></StrictMode>);
