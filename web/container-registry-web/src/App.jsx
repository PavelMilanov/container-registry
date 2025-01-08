// import logo from './logo.svg';
// import styles from './App.module.css';
import { createSignal, Show } from 'solid-js';


import Registry from "./Registry";


function App() {
  const [loggedIn, setLoggedIn] = createSignal(false)
  // const toggle = () => setLoggedIn(!loggedIn())

  return (
    <div>
      <nav>
        <a>Реестры</a>
        <a>Настройки</a>
        <a>Выйти</a>
      </nav>
     <Registry />
    </div>
  );
}

export default App;
