import { createSignal, onMount, lazy } from "solid-js";
import { useNavigate, useParams, useLocation } from "@solidjs/router";
import axios from "axios";
import { showToast } from "./utils/notification";
import Breadcrumb from "./utils/Breadcrumb";
import NavBar from "./NavBar";
const RepoTable = lazy(() => import("./utils/RepoTable"));

export default function Repo() {
  const navigate = useNavigate();
  const location = useLocation();
  const params = useParams();
  const [imageList, setImageList] = createSignal([]);

  async function getRepo() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.get(
        API_URL + `/api/registry/${params.name}`,
        {
          headers: headers,
        },
      );
      setImageList(response.data.data); // в ответе приходит массив "data"
    } catch (error) {
      if (error.response?.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.error(error);
        showToast("Ошибка!", "error");
      }
    }
  }

  onMount(async () => {
    await getRepo();
  });

  return (
    <>
      <NavBar />
      <div class="container">
        <Breadcrumb path={location.pathname} />
        <div class="card">
          <RepoTable
            items={imageList()}
            delNotification={() => {
              getRepo();
            }}
          />
        </div>
      </div>
    </>
  );
}
