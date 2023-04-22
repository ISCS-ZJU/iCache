package rpc

import (
	"encoding/json"
	"main/common"
	"main/rpc/cache"
	"main/services"
	"time"

	log "github.com/sirupsen/logrus"
)

type CacheInfo struct {
	Remote_hit int64 `json:"remote peer hit times"`
}

// get_cache_info: serialize data through JSON
func Grpc_op_imple_get_cache_info(request *cache.DCRequest) (*cache.DCReply, error) {

	// services.DCRuntime.RLock()
	// defer services.DCRuntime.RUnlock()
	var reply cache.DCReply
	reply.Id = services.DCRuntime.Cache_mng.GetRmoteHit()
	return &reply, nil
}

func Grpc_op_imple_readimg_byidx(request *cache.DCRequest) (*cache.DCReply, error) {
	// start := time.Now()

	// services.DCRuntime.RLock()
	// defer services.DCRuntime.RUnlock()

	var reply cache.DCReply
	// if imgidx is not an important sample
	imgidx := request.Imgidx
	fakecache := request.Id
	if fakecache == 1 {
		log.Debug("[deepcache-rpc-impl.go] fake cache == 1")
		// synthetic data; fake cache does NOT fetch real data
		reply.Data = []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0, 32, 0, 0, 0, 32, 8, 2, 0, 0, 0, 252, 24, 237, 163, 0, 0, 6, 13, 73, 68, 65, 84, 120, 156, 93, 86, 203, 118, 27, 215, 17, 172, 234, 190, 51, 0, 68, 240, 33, 145, 150, 163, 80, 39, 217, 120, 17, 219, 127, 144, 252, 77, 62, 36, 171, 252, 94, 86, 73, 214, 142, 79, 114, 20, 133, 226, 19, 196, 107, 102, 110, 119, 101, 113, 65, 136, 246, 221, 160, 49, 103, 166, 187, 111, 85, 245, 131, 127, 249, 235, 159, 1, 24, 13, 132, 153, 185, 59, 105, 102, 4, 152, 153, 146, 164, 140, 170, 102, 40, 69, 73, 96, 194, 170, 24, 41, 161, 29, 101, 102, 196, 4, 65, 18, 9, 7, 36, 9, 42, 238, 78, 210, 204, 0, 152, 153, 153, 145, 108, 1, 72, 0, 144, 140, 204, 140, 72, 153, 152, 76, 165, 64, 200, 141, 52, 74, 2, 144, 41, 33, 9, 9, 2, 36, 65, 20, 13, 144, 74, 115, 77, 178, 133, 105, 134, 4, 64, 237, 57, 8, 7, 1, 82, 74, 18, 2, 33, 0, 102, 68, 38, 32, 65, 66, 16, 97, 84, 42, 91, 86, 45, 57, 18, 165, 229, 217, 252, 226, 152, 54, 112, 180, 9, 242, 128, 24, 1, 177, 144, 34, 105, 130, 153, 73, 145, 169, 164, 137, 130, 192, 164, 41, 83, 32, 13, 146, 64, 148, 87, 254, 121, 244, 174, 23, 108, 95, 162, 194, 221, 72, 24, 1, 194, 192, 128, 77, 145, 164, 74, 97, 13, 133, 210, 8, 22, 7, 216, 16, 18, 83, 16, 164, 66, 218, 235, 27, 72, 122, 49, 18, 0, 105, 228, 33, 140, 185, 193, 204, 101, 213, 115, 59, 150, 205, 52, 33, 199, 243, 185, 119, 172, 38, 4, 11, 15, 236, 89, 166, 18, 209, 216, 46, 175, 9, 104, 90, 1, 104, 6, 233, 144, 187, 153, 181, 136, 36, 9, 50, 226, 121, 95, 255, 251, 184, 125, 115, 118, 217, 207, 231, 67, 125, 58, 45, 180, 116, 170, 147, 133, 209, 32, 7, 4, 33, 21, 4, 75, 75, 240, 72, 239, 43, 160, 242, 53, 118, 47, 136, 69, 198, 176, 93, 197, 162, 108, 114, 187, 90, 79, 182, 156, 211, 78, 223, 162, 24, 32, 129, 0, 33, 3, 196, 246, 79, 42, 238, 126, 244, 115, 12, 211, 188, 53, 167, 102, 102, 230, 237, 102, 0, 188, 91, 188, 191, 92, 46, 222, 239, 251, 254, 246, 223, 255, 90, 19, 23, 86, 206, 97, 128, 82, 89, 36, 66, 135, 26, 50, 16, 100, 33, 45, 83, 175, 139, 224, 181, 144, 218, 243, 226, 29, 128, 108, 49, 124, 62, 236, 214, 171, 255, 253, 237, 143, 127, 138, 31, 191, 255, 240, 207, 191, 15, 143, 15, 163, 178, 183, 12, 161, 207, 196, 65, 31, 4, 4, 146, 165, 81, 193, 95, 31, 188, 144, 77, 51, 55, 115, 210, 76, 9, 118, 187, 168, 55, 171, 159, 46, 47, 203, 111, 223, 47, 175, 46, 183, 183, 55, 203, 231, 167, 62, 177, 36, 119, 130, 29, 56, 164, 140, 214, 242, 107, 1, 248, 58, 125, 30, 216, 252, 170, 46, 17, 197, 156, 180, 96, 60, 62, 223, 207, 78, 234, 245, 135, 183, 111, 79, 103, 117, 255, 249, 205, 236, 162, 248, 172, 98, 70, 153, 148, 102, 1, 64, 121, 192, 83, 82, 201, 76, 119, 127, 173, 250, 102, 30, 43, 65, 4, 92, 70, 115, 195, 102, 124, 126, 218, 62, 189, 91, 250, 213, 219, 187, 113, 136, 253, 70, 219, 245, 179, 49, 204, 7, 139, 55, 105, 147, 25, 128, 20, 73, 80, 66, 100, 148, 22, 231, 208, 204, 50, 127, 197, 129, 36, 38, 56, 9, 125, 36, 203, 195, 83, 53, 149, 197, 108, 154, 205, 183, 183, 119, 171, 187, 219, 105, 187, 127, 83, 108, 33, 21, 131, 147, 41, 229, 65, 238, 217, 10, 93, 197, 91, 211, 250, 42, 196, 95, 120, 7, 4, 185, 37, 216, 235, 113, 179, 254, 233, 231, 79, 224, 167, 235, 247, 183, 176, 231, 161, 134, 205, 60, 87, 39, 224, 155, 194, 158, 172, 164, 73, 158, 153, 74, 201, 14, 1, 14, 222, 143, 231, 165, 214, 208, 154, 96, 166, 148, 65, 100, 21, 62, 127, 249, 148, 250, 114, 125, 189, 186, 254, 184, 43, 158, 169, 222, 202, 197, 56, 94, 209, 122, 99, 255, 90, 122, 71, 151, 102, 44, 191, 204, 23, 199, 134, 113, 108, 120, 82, 202, 181, 218, 76, 55, 55, 255, 57, 63, 255, 242, 195, 143, 118, 186, 220, 175, 30, 202, 20, 101, 156, 22, 49, 125, 132, 39, 131, 4, 147, 199, 143, 216, 166, 139, 164, 2, 124, 101, 252, 53, 74, 237, 42, 6, 130, 80, 193, 205, 231, 251, 221, 254, 246, 251, 31, 166, 190, 219, 220, 223, 62, 110, 239, 174, 54, 123, 247, 238, 234, 100, 246, 27, 250, 100, 153, 210, 87, 15, 205, 62, 244, 162, 136, 145, 52, 32, 0, 154, 153, 91, 167, 195, 75, 9, 37, 1, 247, 50, 238, 226, 238, 233, 31, 31, 62, 142, 134, 233, 211, 207, 251, 97, 191, 216, 61, 21, 234, 242, 155, 111, 255, 32, 153, 203, 193, 90, 51, 35, 162, 13, 193, 23, 68, 4, 176, 212, 58, 145, 204, 108, 117, 225, 102, 81, 188, 115, 243, 151, 161, 33, 7, 158, 215, 247, 230, 43, 34, 110, 62, 69, 198, 137, 80, 230, 221, 213, 229, 197, 119, 198, 111, 106, 12, 76, 2, 153, 249, 50, 97, 219, 180, 202, 131, 81, 34, 39, 128, 46, 147, 32, 85, 178, 134, 167, 211, 220, 96, 238, 242, 66, 214, 145, 15, 165, 248, 118, 213, 119, 121, 225, 126, 66, 247, 139, 247, 191, 239, 250, 119, 195, 72, 90, 109, 168, 70, 196, 17, 88, 64, 153, 7, 209, 151, 246, 147, 130, 164, 8, 17, 145, 137, 36, 228, 164, 58, 55, 159, 106, 60, 239, 158, 98, 226, 217, 226, 154, 249, 174, 148, 69, 205, 93, 215, 159, 78, 145, 144, 252, 192, 26, 95, 64, 103, 171, 167, 118, 155, 54, 50, 117, 44, 181, 67, 71, 203, 74, 34, 4, 144, 16, 54, 187, 241, 238, 97, 248, 112, 245, 225, 116, 126, 181, 219, 206, 187, 226, 26, 247, 181, 14, 70, 151, 60, 194, 91, 132, 99, 181, 30, 73, 6, 16, 153, 69, 168, 16, 5, 18, 109, 230, 176, 197, 75, 9, 169, 41, 243, 246, 254, 121, 62, 255, 246, 226, 236, 210, 210, 246, 5, 197, 77, 94, 166, 97, 156, 207, 22, 18, 4, 101, 230, 81, 126, 47, 237, 224, 171, 246, 75, 141, 60, 244, 52, 136, 52, 182, 161, 46, 147, 12, 44, 171, 231, 237, 118, 183, 253, 221, 245, 123, 131, 155, 193, 48, 208, 172, 235, 103, 227, 56, 45, 230, 5, 68, 102, 214, 90, 37, 89, 91, 166, 64, 40, 37, 2, 84, 166, 32, 203, 68, 132, 50, 85, 51, 106, 78, 66, 128, 34, 205, 216, 103, 240, 238, 238, 254, 236, 236, 244, 164, 135, 50, 72, 235, 139, 39, 115, 182, 92, 142, 201, 125, 164, 204, 0, 19, 152, 66, 164, 106, 32, 18, 41, 166, 152, 178, 154, 170, 153, 165, 237, 95, 41, 18, 56, 94, 77, 148, 27, 158, 159, 30, 179, 142, 23, 167, 75, 40, 4, 139, 228, 80, 245, 184, 94, 159, 158, 149, 245, 46, 215, 195, 243, 188, 148, 179, 147, 147, 210, 45, 76, 26, 198, 58, 197, 161, 211, 181, 98, 136, 176, 136, 90, 162, 45, 88, 104, 43, 201, 177, 229, 65, 168, 210, 244, 238, 98, 57, 47, 70, 132, 192, 64, 9, 118, 67, 29, 176, 171, 179, 197, 249, 242, 244, 180, 64, 38, 209, 75, 70, 181, 174, 79, 196, 88, 167, 58, 213, 200, 148, 48, 142, 83, 230, 84, 18, 109, 141, 96, 91, 5, 14, 50, 64, 166, 116, 126, 118, 98, 214, 89, 6, 12, 41, 122, 55, 123, 179, 60, 185, 42, 103, 103, 231, 231, 17, 65, 242, 226, 236, 52, 166, 113, 28, 198, 97, 218, 202, 124, 54, 159, 151, 72, 44, 80, 235, 244, 244, 244, 4, 243, 76, 43, 106, 171, 222, 139, 12, 154, 18, 34, 131, 148, 91, 49, 36, 179, 130, 94, 19, 155, 245, 190, 204, 150, 25, 57, 110, 247, 230, 62, 78, 227, 205, 126, 52, 247, 168, 33, 88, 199, 34, 192, 221, 251, 174, 219, 15, 92, 44, 230, 251, 253, 182, 86, 21, 162, 113, 15, 210, 208, 182, 82, 165, 148, 164, 8, 210, 44, 162, 2, 26, 167, 248, 242, 176, 58, 57, 99, 239, 54, 141, 251, 174, 120, 14, 251, 41, 49, 212, 236, 251, 206, 189, 236, 119, 251, 90, 167, 90, 167, 136, 28, 134, 253, 172, 47, 99, 157, 54, 219, 85, 65, 43, 98, 33, 204, 18, 112, 102, 27, 72, 162, 36, 139, 4, 97, 82, 236, 39, 174, 55, 155, 180, 249, 188, 88, 183, 92, 156, 244, 115, 114, 74, 228, 188, 235, 220, 162, 78, 67, 86, 17, 54, 14, 235, 205, 102, 87, 188, 76, 185, 27, 211, 188, 227, 255, 1, 37, 240, 159, 193, 70, 254, 183, 5, 0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130}
		reply.Len = 1606
		reply.Clsidx = 0
		reply.Imgidx = 0
		reply.IsHit = true
		reply.Msg = "readimg_byidx FAKE CACHE, IsHit = true"
		return &reply, nil
	} else {
		log.Debugf("[DCSERVER-RPC-IMPLE] request sample: [%v]", imgidx)
		if services.DCRuntime.Cache_type == "isa" || services.DCRuntime.Cache_type == "isa_lru" {
			log.Errorln("when calling readimg_byidx rpc, cache_type should NOT be isa or isa_lru")
		} else if services.DCRuntime.Cache_type == "quiver" {

			// if cache_type is quiver
			// if imgidx == -1: call RandomAccess
			// if imgidx != -1: call Exists: if exists, call Access; otherwise return imgidx == -1
			if imgidx == -1 {
				log.Debug("[DCSERVER-RPC-IMPLE] imgidx is -1, call RandomAccess function")
				// start := time.Now()
				imgcontent, imgidx, _ := services.DCRuntime.Cache_mng.(*(services.Quiver_cache_mng)).RandomAccess()
				for {
					if imgidx == -1 {
						time.Sleep(time.Millisecond)
						imgcontent, imgidx, _ = services.DCRuntime.Cache_mng.(*(services.Quiver_cache_mng)).RandomAccess()
						log.Debugf("Random Access got imgidx is [%v], len(imgcontent) = [%v]", imgidx, len(imgcontent))
					} else {
						log.Debugf("Random Access got imgidx is [%v], len(imgcontent) = [%v]", imgidx, len(imgcontent))
						break
					}
				}
				// elapsed := time.Since(start)
				// log.Infof("-1 hit time access: ", elapsed)
				reply.Data = imgcontent
				reply.Len = int64(len(imgcontent))
				reply.Clsidx = services.DCRuntime.Imgidx2clsidx[imgidx]
				reply.Imgidx = imgidx
				reply.IsHit = true
			} else {

				imgcontent, rt_imgidx := services.DCRuntime.Cache_mng.(*(services.Quiver_cache_mng)).AccessAtOnceQuiver(imgidx)
				if rt_imgidx != -1 {
					reply.Data = imgcontent
					reply.Len = int64(len(imgcontent))
					reply.Clsidx = services.DCRuntime.Imgidx2clsidx[imgidx]
					reply.Imgidx = imgidx
					reply.IsHit = true
				} else {
					reply.Data = nil
					reply.Len = 0
					reply.Clsidx = -1
					reply.Imgidx = -1
					reply.IsHit = false
				}
			}
			reply.Msg = "readimg_byidx"
			return &reply, nil
		}

		imgcontent, clsidx, isHit := services.DCRuntime.Cache_mng.AccessAtOnce(imgidx)

		reply.Data = imgcontent
		reply.Len = int64(len(imgcontent))
		reply.Clsidx = clsidx
		reply.Imgidx = imgidx
		reply.IsHit = isHit
		reply.Msg = "readimg_byidx"

		return &reply, nil
	}

}

func Grpc_op_imple_readimg_byidx_t(request *cache.DCRequest) (*cache.DCReply, error) {
	log.Debugln()
	// services.DCRuntime.RLock()
	// defer services.DCRuntime.RUnlock()

	// get imgidx and is_important from kvmap;
	// kvmap only contains one pair kv
	var reply cache.DCReply
	var kvmap map[int64]int64
	fakecache := request.Id
	if fakecache == 1 {
		log.Debug("[deepcache-rpc-impl.go] fake cache == 1")
		// synthetic data; fake cache does NOT fetch real data
		reply.Data = []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0, 32, 0, 0, 0, 32, 8, 2, 0, 0, 0, 252, 24, 237, 163, 0, 0, 6, 13, 73, 68, 65, 84, 120, 156, 93, 86, 203, 118, 27, 215, 17, 172, 234, 190, 51, 0, 68, 240, 33, 145, 150, 163, 80, 39, 217, 120, 17, 219, 127, 144, 252, 77, 62, 36, 171, 252, 94, 86, 73, 214, 142, 79, 114, 20, 133, 226, 19, 196, 107, 102, 110, 119, 101, 113, 65, 136, 246, 221, 160, 49, 103, 166, 187, 111, 85, 245, 131, 127, 249, 235, 159, 1, 24, 13, 132, 153, 185, 59, 105, 102, 4, 152, 153, 146, 164, 140, 170, 102, 40, 69, 73, 96, 194, 170, 24, 41, 161, 29, 101, 102, 196, 4, 65, 18, 9, 7, 36, 9, 42, 238, 78, 210, 204, 0, 152, 153, 153, 145, 108, 1, 72, 0, 144, 140, 204, 140, 72, 153, 152, 76, 165, 64, 200, 141, 52, 74, 2, 144, 41, 33, 9, 9, 2, 36, 65, 20, 13, 144, 74, 115, 77, 178, 133, 105, 134, 4, 64, 237, 57, 8, 7, 1, 82, 74, 18, 2, 33, 0, 102, 68, 38, 32, 65, 66, 16, 97, 84, 42, 91, 86, 45, 57, 18, 165, 229, 217, 252, 226, 152, 54, 112, 180, 9, 242, 128, 24, 1, 177, 144, 34, 105, 130, 153, 73, 145, 169, 164, 137, 130, 192, 164, 41, 83, 32, 13, 146, 64, 148, 87, 254, 121, 244, 174, 23, 108, 95, 162, 194, 221, 72, 24, 1, 194, 192, 128, 77, 145, 164, 74, 97, 13, 133, 210, 8, 22, 7, 216, 16, 18, 83, 16, 164, 66, 218, 235, 27, 72, 122, 49, 18, 0, 105, 228, 33, 140, 185, 193, 204, 101, 213, 115, 59, 150, 205, 52, 33, 199, 243, 185, 119, 172, 38, 4, 11, 15, 236, 89, 166, 18, 209, 216, 46, 175, 9, 104, 90, 1, 104, 6, 233, 144, 187, 153, 181, 136, 36, 9, 50, 226, 121, 95, 255, 251, 184, 125, 115, 118, 217, 207, 231, 67, 125, 58, 45, 180, 116, 170, 147, 133, 209, 32, 7, 4, 33, 21, 4, 75, 75, 240, 72, 239, 43, 160, 242, 53, 118, 47, 136, 69, 198, 176, 93, 197, 162, 108, 114, 187, 90, 79, 182, 156, 211, 78, 223, 162, 24, 32, 129, 0, 33, 3, 196, 246, 79, 42, 238, 126, 244, 115, 12, 211, 188, 53, 167, 102, 102, 230, 237, 102, 0, 188, 91, 188, 191, 92, 46, 222, 239, 251, 254, 246, 223, 255, 90, 19, 23, 86, 206, 97, 128, 82, 89, 36, 66, 135, 26, 50, 16, 100, 33, 45, 83, 175, 139, 224, 181, 144, 218, 243, 226, 29, 128, 108, 49, 124, 62, 236, 214, 171, 255, 253, 237, 143, 127, 138, 31, 191, 255, 240, 207, 191, 15, 143, 15, 163, 178, 183, 12, 161, 207, 196, 65, 31, 4, 4, 146, 165, 81, 193, 95, 31, 188, 144, 77, 51, 55, 115, 210, 76, 9, 118, 187, 168, 55, 171, 159, 46, 47, 203, 111, 223, 47, 175, 46, 183, 183, 55, 203, 231, 167, 62, 177, 36, 119, 130, 29, 56, 164, 140, 214, 242, 107, 1, 248, 58, 125, 30, 216, 252, 170, 46, 17, 197, 156, 180, 96, 60, 62, 223, 207, 78, 234, 245, 135, 183, 111, 79, 103, 117, 255, 249, 205, 236, 162, 248, 172, 98, 70, 153, 148, 102, 1, 64, 121, 192, 83, 82, 201, 76, 119, 127, 173, 250, 102, 30, 43, 65, 4, 92, 70, 115, 195, 102, 124, 126, 218, 62, 189, 91, 250, 213, 219, 187, 113, 136, 253, 70, 219, 245, 179, 49, 204, 7, 139, 55, 105, 147, 25, 128, 20, 73, 80, 66, 100, 148, 22, 231, 208, 204, 50, 127, 197, 129, 36, 38, 56, 9, 125, 36, 203, 195, 83, 53, 149, 197, 108, 154, 205, 183, 183, 119, 171, 187, 219, 105, 187, 127, 83, 108, 33, 21, 131, 147, 41, 229, 65, 238, 217, 10, 93, 197, 91, 211, 250, 42, 196, 95, 120, 7, 4, 185, 37, 216, 235, 113, 179, 254, 233, 231, 79, 224, 167, 235, 247, 183, 176, 231, 161, 134, 205, 60, 87, 39, 224, 155, 194, 158, 172, 164, 73, 158, 153, 74, 201, 14, 1, 14, 222, 143, 231, 165, 214, 208, 154, 96, 166, 148, 65, 100, 21, 62, 127, 249, 148, 250, 114, 125, 189, 186, 254, 184, 43, 158, 169, 222, 202, 197, 56, 94, 209, 122, 99, 255, 90, 122, 71, 151, 102, 44, 191, 204, 23, 199, 134, 113, 108, 120, 82, 202, 181, 218, 76, 55, 55, 255, 57, 63, 255, 242, 195, 143, 118, 186, 220, 175, 30, 202, 20, 101, 156, 22, 49, 125, 132, 39, 131, 4, 147, 199, 143, 216, 166, 139, 164, 2, 124, 101, 252, 53, 74, 237, 42, 6, 130, 80, 193, 205, 231, 251, 221, 254, 246, 251, 31, 166, 190, 219, 220, 223, 62, 110, 239, 174, 54, 123, 247, 238, 234, 100, 246, 27, 250, 100, 153, 210, 87, 15, 205, 62, 244, 162, 136, 145, 52, 32, 0, 154, 153, 91, 167, 195, 75, 9, 37, 1, 247, 50, 238, 226, 238, 233, 31, 31, 62, 142, 134, 233, 211, 207, 251, 97, 191, 216, 61, 21, 234, 242, 155, 111, 255, 32, 153, 203, 193, 90, 51, 35, 162, 13, 193, 23, 68, 4, 176, 212, 58, 145, 204, 108, 117, 225, 102, 81, 188, 115, 243, 151, 161, 33, 7, 158, 215, 247, 230, 43, 34, 110, 62, 69, 198, 137, 80, 230, 221, 213, 229, 197, 119, 198, 111, 106, 12, 76, 2, 153, 249, 50, 97, 219, 180, 202, 131, 81, 34, 39, 128, 46, 147, 32, 85, 178, 134, 167, 211, 220, 96, 238, 242, 66, 214, 145, 15, 165, 248, 118, 213, 119, 121, 225, 126, 66, 247, 139, 247, 191, 239, 250, 119, 195, 72, 90, 109, 168, 70, 196, 17, 88, 64, 153, 7, 209, 151, 246, 147, 130, 164, 8, 17, 145, 137, 36, 228, 164, 58, 55, 159, 106, 60, 239, 158, 98, 226, 217, 226, 154, 249, 174, 148, 69, 205, 93, 215, 159, 78, 145, 144, 252, 192, 26, 95, 64, 103, 171, 167, 118, 155, 54, 50, 117, 44, 181, 67, 71, 203, 74, 34, 4, 144, 16, 54, 187, 241, 238, 97, 248, 112, 245, 225, 116, 126, 181, 219, 206, 187, 226, 26, 247, 181, 14, 70, 151, 60, 194, 91, 132, 99, 181, 30, 73, 6, 16, 153, 69, 168, 16, 5, 18, 109, 230, 176, 197, 75, 9, 169, 41, 243, 246, 254, 121, 62, 255, 246, 226, 236, 210, 210, 246, 5, 197, 77, 94, 166, 97, 156, 207, 22, 18, 4, 101, 230, 81, 126, 47, 237, 224, 171, 246, 75, 141, 60, 244, 52, 136, 52, 182, 161, 46, 147, 12, 44, 171, 231, 237, 118, 183, 253, 221, 245, 123, 131, 155, 193, 48, 208, 172, 235, 103, 227, 56, 45, 230, 5, 68, 102, 214, 90, 37, 89, 91, 166, 64, 40, 37, 2, 84, 166, 32, 203, 68, 132, 50, 85, 51, 106, 78, 66, 128, 34, 205, 216, 103, 240, 238, 238, 254, 236, 236, 244, 164, 135, 50, 72, 235, 139, 39, 115, 182, 92, 142, 201, 125, 164, 204, 0, 19, 152, 66, 164, 106, 32, 18, 41, 166, 152, 178, 154, 170, 153, 165, 237, 95, 41, 18, 56, 94, 77, 148, 27, 158, 159, 30, 179, 142, 23, 167, 75, 40, 4, 139, 228, 80, 245, 184, 94, 159, 158, 149, 245, 46, 215, 195, 243, 188, 148, 179, 147, 147, 210, 45, 76, 26, 198, 58, 197, 161, 211, 181, 98, 136, 176, 136, 90, 162, 45, 88, 104, 43, 201, 177, 229, 65, 168, 210, 244, 238, 98, 57, 47, 70, 132, 192, 64, 9, 118, 67, 29, 176, 171, 179, 197, 249, 242, 244, 180, 64, 38, 209, 75, 70, 181, 174, 79, 196, 88, 167, 58, 213, 200, 148, 48, 142, 83, 230, 84, 18, 109, 141, 96, 91, 5, 14, 50, 64, 166, 116, 126, 118, 98, 214, 89, 6, 12, 41, 122, 55, 123, 179, 60, 185, 42, 103, 103, 231, 231, 17, 65, 242, 226, 236, 52, 166, 113, 28, 198, 97, 218, 202, 124, 54, 159, 151, 72, 44, 80, 235, 244, 244, 244, 4, 243, 76, 43, 106, 171, 222, 139, 12, 154, 18, 34, 131, 148, 91, 49, 36, 179, 130, 94, 19, 155, 245, 190, 204, 150, 25, 57, 110, 247, 230, 62, 78, 227, 205, 126, 52, 247, 168, 33, 88, 199, 34, 192, 221, 251, 174, 219, 15, 92, 44, 230, 251, 253, 182, 86, 21, 162, 113, 15, 210, 208, 182, 82, 165, 148, 164, 8, 210, 44, 162, 2, 26, 167, 248, 242, 176, 58, 57, 99, 239, 54, 141, 251, 174, 120, 14, 251, 41, 49, 212, 236, 251, 206, 189, 236, 119, 251, 90, 167, 90, 167, 136, 28, 134, 253, 172, 47, 99, 157, 54, 219, 85, 65, 43, 98, 33, 204, 18, 112, 102, 27, 72, 162, 36, 139, 4, 97, 82, 236, 39, 174, 55, 155, 180, 249, 188, 88, 183, 92, 156, 244, 115, 114, 74, 228, 188, 235, 220, 162, 78, 67, 86, 17, 54, 14, 235, 205, 102, 87, 188, 76, 185, 27, 211, 188, 227, 255, 1, 37, 240, 159, 193, 70, 254, 183, 5, 0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130}
		reply.Len = 1606
		reply.Clsidx = 0
		reply.Imgidx = 0
		reply.IsHit = true
		reply.Msg = "readimg_byidx FAKE CACHE, IsHit = true"
		return &reply, nil

	} else {
		err := json.Unmarshal([]byte(request.ImgidxT), &kvmap)
		if err != nil {
			log.Debug("[DCSERVER-RPC-IMPLE] json err:", err)
		}
		full_access := request.FullAccess
		log.Debug("[DCSERVER-RPC-IMPLE] parse imgidx_t done")
		is_important := false
		imgidx := int64(0)
		for key := range kvmap {
			imgidx = key
			is_important = (kvmap[imgidx] == 1)
		}
		log.Debugf("[DCSERVER-RPC-IMPLE] request sample: [%v]", imgidx)

		if services.DCRuntime.Cache_type == "isa" && (!is_important) && common.Config.Package_design {
			log.Debug("[DCSERVER-RPC-IMPLE] is unimportant and going to UC")
			reply.Data, reply.Imgidx, _ = services.DCRuntime.Cache_mng.(*services.Isa_cache_mng).Pack_metadata.RandomAccess()
			log.Debugf("[DCSERVER-RPC-IMPLE] got reply [%v] from RandomAccess", reply.Imgidx)
			// in case that no data in sequencial buffer
			if reply.Imgidx == -1 {
				log.Info("[DCSERVER-RPC-IMPLE] return -1 from UC, you'd better enlarge UC size.")
				reply.Data = nil
				reply.Len = 0
				reply.Clsidx = -1
				reply.Imgidx = -1
				reply.IsHit = false
				return &reply, nil
			}
			reply.Len = int64(len(reply.Data))
			reply.Clsidx = services.DCRuntime.Imgidx2clsidx[reply.Imgidx]
			reply.Msg = "readimg_byidx"
			reply.IsHit = true
			log.Debugf("[DCSERVER-RPC-IMPLE] Call RandomAccess done and get imgidx [%v]", reply.Imgidx)
			return &reply, nil
		} else if services.DCRuntime.Cache_type == "isa" && full_access {
			// if cache is full, not need to insert into cache, so use AccessAtOnceISA
			imgcontent, clsidx, isHit := services.DCRuntime.Cache_mng.(*(services.Isa_cache_mng)).AccessAtOnceISA(imgidx)

			reply.Data = imgcontent
			reply.Len = int64(len(imgcontent))
			reply.Clsidx = clsidx
			reply.Imgidx = imgidx
			reply.IsHit = isHit
			reply.Msg = "readimg_byidx"

			return &reply, nil
		} else {
			// log.Info("xxxxxxxxxx")
			imgcontent, clsidx, isHit := services.DCRuntime.Cache_mng.AccessAtOnce(imgidx)
			// log.Info("donexxxxx")

			reply.Data = imgcontent
			reply.Len = int64(len(imgcontent))
			reply.Clsidx = clsidx
			reply.Imgidx = imgidx
			reply.IsHit = isHit
			reply.Msg = "readimg_byidx"

			return &reply, nil
		}
	}

}

func Grpc_op_imple_update_ivpersample(request *cache.DCRequest) (*cache.DCReply, error) {
	log.Debug("[DCSERVER-RPC-IMPLE] kvlist_json[0]:", request.KvlistJson[0]) // for map struct
	var reply cache.DCReply
	if services.DCRuntime.Cache_type == "isa" {
		// isa
		// parse kvlist from request
		var kvmap map[int64]float64
		err := json.Unmarshal([]byte(request.KvlistJson), &kvmap)
		if err != nil {
			log.Debug("[DCSERVER-RPC-IMPLE] json err:", err)
		}
		log.Debug("[DCSERVER-RPC-IMPLE] parse kvlist_json done")

		services.DCRuntime.Isa_related_metadata.Lock()
		log.Debug("[DCSERVER-RPC-IMPLE] get Isa_related_metadata.Lock()")
		defer services.DCRuntime.Isa_related_metadata.Unlock()

		// update kv, key:imgidx value: new importance value
		for imgidx, importance := range kvmap {
			services.DCRuntime.Isa_related_metadata.New_Ivpersample[imgidx] = importance
		}

		services.DCRuntime.Cache_mng.(*services.Isa_cache_mng).Sent_async_build_shadow_heap_task(services.By_Map, kvmap, 0, 0, 0)

	}
	reply.Msg = "update_ivpersample-2"
	return &reply, nil
}

func Grpc_op_imple_refresh_server_ivpsample(request *cache.DCRequest) (*cache.DCReply, error) {

	services.DCRuntime.Isa_related_metadata.Lock()
	log.Debug("[DCSERVER-RPC-IMPLE] get Isa_related_metadata.Lock()")
	defer services.DCRuntime.Isa_related_metadata.Unlock()

	var reply cache.DCReply
	if services.DCRuntime.Cache_type == "isa" {
		services.DCRuntime.Cache_mng.(*services.Isa_cache_mng).Lock()
		services.DCRuntime.Cache_mng.(*services.Isa_cache_mng).Sent_async_build_shadow_heap_task(services.SWITCH_HEAP, nil, 0, 0, 0)
		<-services.DCRuntime.Cache_mng.(*services.Isa_cache_mng).Switching_shadow_channel
		// services.DCRuntime.Cache_mng.(*services.Isa_cache_mng).Switch_to_ShadowHeap()
		// update Ivpersample
		mp := make(map[int64]float64)
		for imgidx, impvalue := range services.DCRuntime.Isa_related_metadata.New_Ivpersample {
			mp[imgidx] = impvalue
		}
		services.DCRuntime.Isa_related_metadata.Ivpersample = mp
		services.DCRuntime.Cache_mng.(*services.Isa_cache_mng).Unlock()
		log.Debug("=> after refresh ivpersample len:", len(services.DCRuntime.Isa_related_metadata.Ivpersample))
		// for i := int64(0); i < 20; i++ {
		// 	log.Infoln(services.DCRuntime.Isa_related_metadata.Ivpersample[i])
		// }
		log.Infoln("=> refresh ivpersample done")
	}

	reply.Msg = "update_ivpersample-2"
	return &reply, nil
}
