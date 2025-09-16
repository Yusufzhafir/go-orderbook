import { LoadingSpinner } from "../ui/loadingSpinner";

const LoadingFallback = () => {
    return (
        <div className="grid w-screen h-screen place-items-center">
            <div className="flex gap-2 p-2 border-b border-black">
                loading ...<span><LoadingSpinner /></span>
            </div>
        </div>
    );
}

export default LoadingFallback;