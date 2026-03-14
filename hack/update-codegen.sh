# 定义 code-generator 根路径
CODEGEN_PATH="/root/code-generator"

# v0.35.0 deepcopy-gen 官方命令（唯一正确写法）
${CODEGEN_PATH}/bin/deepcopy-gen ./apis/example.com/v1beta1

# 核心命令（仅指定包路径和输出包名，无自定义参数）
#${CODEGEN_PATH}/bin/client-gen --clientset-name versioned ./apis/example.com/v1beta1