import { lazy } from "solid-js"
import { Router, Route } from "@solidjs/router"

import Registry from "./Registry"

const Register = lazy(() => import("./Register"))
const Repo = lazy(() => import("./Repo"))
const Image = lazy(() => import("./Image"))
const Login = lazy(() => import("./Login"))
const Logout = lazy(() => import("./modal/Logout"))
const NotFound = lazy(() => import("./NotFound"))

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
        <Route path="*" component={NotFound} />
      </Router>
    </div>
  )
}

export default App
