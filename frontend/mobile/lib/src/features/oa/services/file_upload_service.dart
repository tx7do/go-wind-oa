import 'package:dio/dio.dart' show DioException, FormData, MultipartFile;
import 'package:get_it/get_it.dart' show GetIt;
import 'package:dio/dio.dart' show Dio;
import 'package:image_picker/image_picker.dart' show XFile;

import 'package:flutter_app/src/core/services/base_service.dart';
import 'package:flutter_app/src/core/transport/http/status.dart';

/// 发票文件上传结果。
class UploadedInvoice {
  final int fileId;
  final String objectUrl;

  const UploadedInvoice({required this.fileId, required this.objectUrl});
}

/// 文件上传服务（multipart 流式）。
///
/// 报销发票凭证：拍一张/选一张 → multipart POST /app/v1/file/upload →
/// 返回落库的 fileId（报销明细 invoiceFileId 引用）与对象下载地址。
/// 上传经全局注册的 Dio（带鉴权拦截器），失败返回 [Status]。
class FileUploadService extends BaseService {
  FileUploadService() : super(tag: 'FileUploadService');

  Dio get _dio => GetIt.instance<Dio>();

  /// 上传发票图片。fileName 为展示用的原始文件名。
  Future<dynamic> uploadInvoice(XFile image, {String? fileName}) async {
    try {
      final form = FormData.fromMap({
        'file': await MultipartFile.fromFile(
          image.path,
          filename: fileName ?? image.name,
        ),
        'sourceFileName': fileName ?? image.name,
      });
      final resp = await _dio.post('/app/v1/file/upload', data: form);
      final data = resp.data;
      if (data is Map<String, dynamic>) {
        final fileId = data['fileId'];
        if (fileId != null && fileId != 0) {
          return UploadedInvoice(
            fileId: fileId is int ? fileId : int.tryParse('$fileId') ?? 0,
            objectUrl: '${data['objectName'] ?? ''}',
          );
        }
      }
      return Status(code: 0, reason: 'upload', message: '上传响应缺少 fileId');
    } on DioException catch (e) {
      return handleDioError(e);
    }
  }
}
