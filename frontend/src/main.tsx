import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './index.css';

// Точка входа React-приложения: подключает корневой компонент к div#root из
// index.html. StrictMode помогает поймать побочные эффекты в разработке.
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
