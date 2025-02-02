import { lazy } from "solid-js"
import { Router, Route, redirect } from "@solidjs/router"

import Registry from "./Registry"

const Register = lazy(() => import("./Register"))
const Repo = lazy(() => import("./Repo"))
const Image = lazy(() => import("./Image"))
const Login = lazy(() => import("./Login"))
const Logout = lazy(() => import("./modal/Logout"))

function App() {
  
  return (
    <div>
      <Router>
        <Route path="/login" component={Login} />
        <Route path="/logout" component={Logout}/>
        <Route path="/register" component={Register} />
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
