import { createSignal, onMount, lazy } from "solid-js";
import { A, useParams, useLocation, useNavigate } from "@solidjs/router";
import axios from "axios";
import { showToast } from "./utils/notification";
import Breadcrumb from "./utils/Breadcrumb";

const ImageTable = lazy(() => import("./utils/ImageTable"));
const Delete = lazy(() => import("./modal/Delete"));

export default function Image() {
  const navigate = useNavigate();
  const params = useParams();
  const location = useLocation();
  const [tagList, setTagList] = createSignal([]);
  const [tag, setTag] = createSignal("");
  const [image, setImage] = createSignal("");
  const [isModalDeleteOpen, setModalDeleteOpen] = createSignal(false);
  const openDeleteModal = (img, tag) => {
    setModalDeleteOpen(true);
    setImage(img);
    setTag(tag);
  };
  const closeDeleteModal = () => setModalDeleteOpen(false);

  const submitDelete = async () => {
    setModalDeleteOpen(false);
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.delete(
        API_URL + `/api/registry/${params.name}/${image()}`,
        { headers: headers, params: { tag: tag() } },
      );
      if (response.status == 202) {
        showToast("Образ удален!");
      }
      await getImages();
    } catch (error) {
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.error(error);
        showToast("Ошибка!", "error");
      }
    }
  };

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
    <div class="container">
      <Breadcrumb path={location.pathname} />
      <div class="card">
        <Delete
          isOpen={isModalDeleteOpen()}
          message={"Образ Docker будет удален!"}
          onClose={closeDeleteModal}
          onSubmit={submitDelete}
        />
        <ImageTable items={tagList()} />
      </div>
    </div>
  );
}
