import { createSignal, onMount, lazy } from "solid-js";
import { useParams, useLocation, useNavigate } from "@solidjs/router";
import axios from "axios";
import { showToast } from "./utils/notification";
import Breadcrumb from "./utils/Breadcrumb";
import NavBar from "./NavBar";
const ImageTable = lazy(() => import("./utils/ImageTable"));

export default function Image() {
  const navigate = useNavigate();
  const params = useParams();
  const location = useLocation();
  const [tagList, setTagList] = createSignal([]);

  async function getImages() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.get(
        API_URL + `/api/registry/${params.name}/${params.image}`,
        { headers: headers },
      );
      setTagList(response.data.data); // в ответе приходит массив "data"
    } catch (error) {
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.error(error);
        showToast("Ошибка!", "error");
      }
    }
  }

  onMount(async () => {
    await getImages();
  });
  return (
    <>
      <NavBar />
      <div class="container">
        <Breadcrumb path={location.pathname} />
        <div class="card">
          <ImageTable
            items={tagList()}
            delNotification={() => {
              getImages();
            }}
          />
        </div>
      </div>
    </>
  );
}
