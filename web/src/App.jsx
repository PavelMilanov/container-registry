import { lazy } from "solid-js"
import { Router, Route } from "@solidjs/router"

import Registry from "./Registry"

const Repo = lazy(() => import("./Repo"))
const Image = lazy(() => import("./Image"))
const Login = lazy(() => import("./Login"))

function App() {
  
  return (
    <div>
      <Router>
        <Route path="/login" component={Login} />
        <Route path="/registry">
          <Route path="/" component={Registry}/>
          <Route path="/:name" component={Repo} />
          <Route path="/:name/:image" component={Image} />
        </Route>
      </Router>
    </div>
  )
}

export default App
