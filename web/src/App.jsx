import { lazy } from "solid-js";
import { Router, Route } from "@solidjs/router";
import { Toaster } from "solid-toast";

import Registry from "./Registry";
import NavBar from "./NavBar";
import GithubLink from "./utils/GithubLink";
const Register = lazy(() => import("./Register"));
const Repo = lazy(() => import("./Repo"));
const Image = lazy(() => import("./Image"));
const Login = lazy(() => import("./Login"));
const Logout = lazy(() => import("./modal/Logout"));
const NotFound = lazy(() => import("./NotFound"));
const Settings = lazy(() => import("./Settings"));

export default function App() {
  return (
    <div>
      <NavBar />
      <Router>
        <Route path="/login" component={Login} />
        <Route path="/logout" component={Logout} />
        <Route path="/register" component={Register} />
        <Route path="/registry">
          <Route path="/" component={Registry} />
          <Route path="/:name" component={Repo} />
          <Route path="/:name/:image" component={Image} />
        </Route>
        <Route path="/settings" component={Settings} />
        <Route path="*" component={NotFound} />
      </Router>
      <GithubLink />
      <Toaster />
    </div>
  );
}
